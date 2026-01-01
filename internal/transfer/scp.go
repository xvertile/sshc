// Package transfer provides file transfer functionality using SCP/SFTP
package transfer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Direction represents the transfer direction
type Direction int

const (
	Upload Direction = iota
	Download
)

func (d Direction) String() string {
	switch d {
	case Upload:
		return "Upload"
	case Download:
		return "Download"
	default:
		return "Unknown"
	}
}

// TransferRequest represents a file transfer request
type TransferRequest struct {
	Host       string    // SSH host name from config
	Direction  Direction // Upload or Download
	LocalPath  string    // Local file/directory path
	RemotePath string    // Remote file/directory path
	Recursive  bool      // Transfer directories recursively
	ConfigFile string    // Optional SSH config file path
}

// TransferResult represents the result of a transfer operation
type TransferResult struct {
	Success   bool
	BytesSent int64
	Error     error
}

// ParseTransferArgs parses scp-style arguments into a TransferRequest
// Examples:
//   - "./local.txt", "host:/remote/path" -> Upload
//   - "host:/remote/file.txt", "./local/" -> Download
func ParseTransferArgs(source, dest string) (*TransferRequest, error) {
	sourceHasHost := strings.Contains(source, ":")
	destHasHost := strings.Contains(dest, ":")

	if sourceHasHost && destHasHost {
		return nil, fmt.Errorf("cannot transfer between two remote hosts")
	}

	if !sourceHasHost && !destHasHost {
		return nil, fmt.Errorf("either source or destination must be a remote path (host:/path)")
	}

	req := &TransferRequest{}

	if sourceHasHost {
		// Download: host:/path -> local
		req.Direction = Download
		parts := strings.SplitN(source, ":", 2)
		req.Host = parts[0]
		req.RemotePath = parts[1]
		req.LocalPath = dest
	} else {
		// Upload: local -> host:/path
		req.Direction = Upload
		parts := strings.SplitN(dest, ":", 2)
		req.Host = parts[0]
		req.RemotePath = parts[1]
		req.LocalPath = source
	}

	// Validate local path exists for uploads
	if req.Direction == Upload {
		info, err := os.Stat(req.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("local path does not exist: %s", req.LocalPath)
		}
		if info.IsDir() {
			req.Recursive = true
		}
	}

	return req, nil
}

// BuildSCPCommand builds the scp command for the transfer
func (r *TransferRequest) BuildSCPCommand() *exec.Cmd {
	args := []string{}

	// Add recursive flag if needed
	if r.Recursive {
		args = append(args, "-r")
	}

	// Add config file if specified
	if r.ConfigFile != "" {
		args = append(args, "-F", r.ConfigFile)
	}

	// Build source and destination based on direction
	var source, dest string
	if r.Direction == Upload {
		source = r.LocalPath
		dest = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
	} else {
		source = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
		dest = r.LocalPath
	}

	args = append(args, source, dest)

	return exec.Command("scp", args...)
}

// Execute runs the transfer and returns the result
func (r *TransferRequest) Execute() *TransferResult {
	cmd := r.BuildSCPCommand()

	// Connect stdin/stdout/stderr for interactive use (password prompts, etc.)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return &TransferResult{
			Success: false,
			Error:   err,
		}
	}

	return &TransferResult{
		Success: true,
	}
}

// ExecuteWithProgress runs the transfer with progress callback
// This uses scp's built-in progress indicator
func (r *TransferRequest) ExecuteWithProgress() *TransferResult {
	args := []string{}

	// Add recursive flag if needed
	if r.Recursive {
		args = append(args, "-r")
	}

	// Add config file if specified
	if r.ConfigFile != "" {
		args = append(args, "-F", r.ConfigFile)
	}

	// Build source and destination based on direction
	var source, dest string
	if r.Direction == Upload {
		source = r.LocalPath
		dest = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
	} else {
		source = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
		dest = r.LocalPath
	}

	args = append(args, source, dest)

	cmd := exec.Command("scp", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return &TransferResult{
			Success: false,
			Error:   err,
		}
	}

	return &TransferResult{
		Success: true,
	}
}

// RunningTransfer represents a transfer that can be cancelled
type RunningTransfer struct {
	cmd    *exec.Cmd
	done   chan *TransferResult
	killed bool
}

// StartTransfer starts a transfer and returns a RunningTransfer that can be cancelled
func (r *TransferRequest) StartTransfer() *RunningTransfer {
	args := []string{}

	// Add recursive flag if needed
	if r.Recursive {
		args = append(args, "-r")
	}

	// Add config file if specified
	if r.ConfigFile != "" {
		args = append(args, "-F", r.ConfigFile)
	}

	// Build source and destination based on direction
	var source, dest string
	if r.Direction == Upload {
		source = r.LocalPath
		dest = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
	} else {
		source = fmt.Sprintf("%s:%s", r.Host, r.RemotePath)
		dest = r.LocalPath
	}

	args = append(args, source, dest)

	cmd := exec.Command("scp", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	rt := &RunningTransfer{
		cmd:  cmd,
		done: make(chan *TransferResult, 1),
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		rt.done <- &TransferResult{Success: false, Error: err}
		return rt
	}

	// Wait for completion in goroutine
	go func() {
		err := cmd.Wait()
		if rt.killed {
			rt.done <- &TransferResult{Success: false, Error: fmt.Errorf("transfer cancelled")}
		} else if err != nil {
			rt.done <- &TransferResult{Success: false, Error: err}
		} else {
			rt.done <- &TransferResult{Success: true}
		}
	}()

	return rt
}

// Cancel kills the running transfer
func (rt *RunningTransfer) Cancel() {
	if rt.cmd != nil && rt.cmd.Process != nil {
		rt.killed = true
		rt.cmd.Process.Kill()
	}
}

// Done returns a channel that receives the result when transfer completes
func (rt *RunningTransfer) Done() <-chan *TransferResult {
	return rt.done
}

// ValidateLocalPath checks if a local path is valid for the given direction
func ValidateLocalPath(path string, direction Direction) error {
	if direction == Upload {
		// For uploads, path must exist
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("file or directory does not exist: %s", path)
		}
	} else {
		// For downloads, parent directory must exist
		parent := filepath.Dir(path)
		if parent != "." && parent != "" {
			if _, err := os.Stat(parent); err != nil {
				return fmt.Errorf("destination directory does not exist: %s", parent)
			}
		}
	}
	return nil
}

// ExpandPath expands ~ to home directory and makes path absolute
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	// Make absolute if relative
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}

	return path, nil
}

// GetLocalFiles returns a list of files/directories in the given path
func GetLocalFiles(path string) ([]FileInfo, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  info.Size(),
		})
	}

	return files, nil
}

// FileInfo represents basic file information
type FileInfo struct {
	Name  string
	IsDir bool
	Size  int64
}
