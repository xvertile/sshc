package transfer

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// RemoteFile represents a file on the remote server
type RemoteFile struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime string
}

// SFTPSession manages an SFTP connection for browsing
type SFTPSession struct {
	client     *ssh.Client
	host       string
	configFile string
}

// NewSFTPSession creates a new SFTP session using SSH agent
func NewSFTPSession(host, configFile string) (*SFTPSession, error) {
	// Get SSH agent connection
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, fmt.Errorf("SSH agent not available (SSH_AUTH_SOCK not set)")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)

	// Get signers from agent
	signers, err := agentClient.Signers()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get signers from SSH agent: %w", err)
	}

	if len(signers) == 0 {
		conn.Close()
		return nil, fmt.Errorf("no keys available in SSH agent")
	}

	// Create SSH config
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
	}

	// Parse host to get actual hostname and port
	// The host is an SSH config alias, so we need to resolve it
	hostname, port, user := resolveSSHHost(host, configFile)
	if user != "" {
		config.User = user
	}

	addr := fmt.Sprintf("%s:%s", hostname, port)

	// Connect
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &SFTPSession{
		client:     client,
		host:       host,
		configFile: configFile,
	}, nil
}

// resolveSSHHost resolves an SSH config alias to hostname, port, and user
func resolveSSHHost(host, configFile string) (hostname, port, user string) {
	// Default values
	hostname = host
	port = "22"
	user = os.Getenv("USER")

	// Try to resolve using ssh -G
	args := []string{"-G", host}
	if configFile != "" {
		args = []string{"-F", configFile, "-G", host}
	}

	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	// Parse the output
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		value := parts[1]

		switch key {
		case "hostname":
			hostname = value
		case "port":
			port = value
		case "user":
			user = value
		}
	}

	return
}

// ListDirectory lists files in a remote directory
func (s *SFTPSession) ListDirectory(path string) ([]RemoteFile, error) {
	// Use SSH to list directory since we're not using full SFTP library
	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Expand ~ to home directory
	if strings.HasPrefix(path, "~") {
		// Get home directory
		homeSession, err := s.client.NewSession()
		if err == nil {
			homeOutput, err := homeSession.Output("echo $HOME")
			homeSession.Close()
			if err == nil {
				home := strings.TrimSpace(string(homeOutput))
				path = strings.Replace(path, "~", home, 1)
			}
		}
	}

	// List directory with details
	cmd := fmt.Sprintf("ls -la %q 2>/dev/null | tail -n +2", path)
	output, err := session.Output(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var files []RemoteFile

	// Add parent directory entry
	if path != "/" {
		files = append(files, RemoteFile{
			Name:  "..",
			Path:  filepath.Dir(path),
			IsDir: true,
		})
	}

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse ls -la output
		// drwxr-xr-x  2 user group  4096 Jan  1 12:00 dirname
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		permissions := fields[0]
		size := int64(0)
		fmt.Sscanf(fields[4], "%d", &size)
		name := strings.Join(fields[8:], " ")

		// Skip . and .. entries from ls output
		if name == "." || name == ".." {
			continue
		}

		isDir := strings.HasPrefix(permissions, "d")
		isLink := strings.HasPrefix(permissions, "l")

		// Handle symlinks
		if isLink {
			// Check if link points to directory
			checkSession, err := s.client.NewSession()
			if err == nil {
				linkPath := filepath.Join(path, name)
				checkCmd := fmt.Sprintf("test -d %q && echo dir", linkPath)
				checkOutput, _ := checkSession.Output(checkCmd)
				checkSession.Close()
				isDir = strings.TrimSpace(string(checkOutput)) == "dir"
			}
		}

		files = append(files, RemoteFile{
			Name:  name,
			Path:  filepath.Join(path, name),
			IsDir: isDir,
			Size:  size,
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].Name == ".." {
			return true
		}
		if files[j].Name == ".." {
			return false
		}
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// GetHomeDirectory returns the remote home directory
func (s *SFTPSession) GetHomeDirectory() (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.Output("echo $HOME")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Close closes the SFTP session
func (s *SFTPSession) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// ReadFile reads a remote file (for small files only)
func (s *SFTPSession) ReadFile(path string, w io.Writer) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = w
	return session.Run(fmt.Sprintf("cat %q", path))
}

// Stat returns file info for a remote path
func (s *SFTPSession) Stat(path string) (*RemoteFile, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("ls -ld %q 2>/dev/null", path)
	output, err := session.Output(cmd)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %s", path)
	}

	line := strings.TrimSpace(string(output))
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nil, fmt.Errorf("unexpected ls output")
	}

	permissions := fields[0]
	size := int64(0)
	fmt.Sscanf(fields[4], "%d", &size)
	name := filepath.Base(path)

	return &RemoteFile{
		Name:  name,
		Path:  path,
		IsDir: strings.HasPrefix(permissions, "d"),
		Size:  size,
	}, nil
}

// HasLocate checks if locate/mlocate is available on the remote system
func (s *SFTPSession) HasLocate() bool {
	session, err := s.client.NewSession()
	if err != nil {
		return false
	}
	defer session.Close()

	err = session.Run("which locate >/dev/null 2>&1 || which mlocate >/dev/null 2>&1")
	return err == nil
}

// Search searches for files matching the pattern
// Uses locate if available (fast, indexed), otherwise falls back to find
func (s *SFTPSession) Search(pattern, startDir string, limit int) ([]RemoteFile, error) {
	if limit <= 0 {
		limit = 100
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Expand ~ in startDir
	if strings.HasPrefix(startDir, "~") {
		homeSession, err := s.client.NewSession()
		if err == nil {
			homeOutput, err := homeSession.Output("echo $HOME")
			homeSession.Close()
			if err == nil {
				home := strings.TrimSpace(string(homeOutput))
				startDir = strings.Replace(startDir, "~", home, 1)
			}
		}
	}

	// Build search command
	// Try to use fd (fast), then find
	// Pattern matching: *pattern* for glob-style matching
	var cmd string

	// First check if fd is available (much faster than find)
	fdCheck, _ := s.client.NewSession()
	hasFd := fdCheck.Run("which fd >/dev/null 2>&1") == nil
	fdCheck.Close()

	if hasFd {
		// fd is super fast and has nice defaults
		cmd = fmt.Sprintf("fd -H -I --max-results %d %q %q 2>/dev/null", limit, pattern, startDir)
	} else {
		// Fall back to find with iname for case-insensitive matching
		cmd = fmt.Sprintf("find %q -iname '*%s*' 2>/dev/null | head -n %d", startDir, pattern, limit)
	}

	output, err := session.Output(cmd)
	if err != nil {
		// Search might return no results, which is not an error
		return []RemoteFile{}, nil
	}

	var files []RemoteFile
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Get file info
		infoSession, err := s.client.NewSession()
		if err != nil {
			continue
		}

		infoCmd := fmt.Sprintf("ls -ld %q 2>/dev/null", line)
		infoOutput, err := infoSession.Output(infoCmd)
		infoSession.Close()

		if err != nil {
			// File might not exist anymore
			continue
		}

		infoLine := strings.TrimSpace(string(infoOutput))
		fields := strings.Fields(infoLine)
		if len(fields) < 9 {
			continue
		}

		permissions := fields[0]
		size := int64(0)
		fmt.Sscanf(fields[4], "%d", &size)

		files = append(files, RemoteFile{
			Name:  filepath.Base(line),
			Path:  line,
			IsDir: strings.HasPrefix(permissions, "d"),
			Size:  size,
		})
	}

	return files, nil
}

// QuickSearch does a faster search without fetching file details
// Uses timeout and depth limit to avoid slow searches
func (s *SFTPSession) QuickSearch(pattern, startDir string, limit int) ([]RemoteFile, error) {
	if limit <= 0 {
		limit = 30
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// Expand ~ in startDir
	if strings.HasPrefix(startDir, "~") {
		homeSession, err := s.client.NewSession()
		if err == nil {
			homeOutput, err := homeSession.Output("echo $HOME")
			homeSession.Close()
			if err == nil {
				home := strings.TrimSpace(string(homeOutput))
				startDir = strings.Replace(startDir, "~", home, 1)
			}
		}
	}

	// Sanitize pattern to prevent command injection
	pattern = strings.ReplaceAll(pattern, "'", "")
	pattern = strings.ReplaceAll(pattern, "\"", "")
	pattern = strings.ReplaceAll(pattern, ";", "")
	pattern = strings.ReplaceAll(pattern, "|", "")
	pattern = strings.ReplaceAll(pattern, "&", "")
	pattern = strings.ReplaceAll(pattern, "$", "")
	pattern = strings.ReplaceAll(pattern, "`", "")

	// Use find with depth limit and timeout for faster results
	// -maxdepth 5 limits how deep we search
	// timeout 3s kills the search after 3 seconds
	cmd := fmt.Sprintf("timeout 3s find %q -maxdepth 5 -iname '*%s*' -printf '%%y %%p\\n' 2>/dev/null | head -n %d", startDir, pattern, limit)

	output, err := session.Output(cmd)
	if err != nil {
		// Try simpler find without -printf and timeout (BSD/macOS compatibility)
		session2, _ := s.client.NewSession()
		// macOS uses gtimeout (from coreutils) or we skip timeout
		cmd = fmt.Sprintf("find %q -maxdepth 5 -iname '*%s*' 2>/dev/null | head -n %d | while read f; do if [ -d \"$f\" ]; then echo \"d $f\"; else echo \"f $f\"; fi; done", startDir, pattern, limit)
		output, err = session2.Output(cmd)
		session2.Close()
		if err != nil {
			return []RemoteFile{}, nil
		}
	}

	var files []RemoteFile
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || len(line) < 3 {
			continue
		}

		typeChar := line[0]
		path := strings.TrimSpace(line[2:])

		files = append(files, RemoteFile{
			Name:  filepath.Base(path),
			Path:  path,
			IsDir: typeChar == 'd',
		})
	}

	return files, nil
}
