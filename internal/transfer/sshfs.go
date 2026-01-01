package transfer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// SSHFSMount represents a mounted SSHFS filesystem
type SSHFSMount struct {
	Host       string
	RemotePath string
	MountPoint string
	ConfigFile string
}

// IsSSHFSAvailable checks if SSHFS is installed
func IsSSHFSAvailable() bool {
	_, err := exec.LookPath("sshfs")
	return err == nil
}

// GetSSHFSInstallInstructions returns platform-specific install instructions
func GetSSHFSInstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "Install with: brew install macfuse sshfs\n(Requires restart after installing macFUSE)"
	case "linux":
		return "Install with: sudo apt install sshfs (Debian/Ubuntu) or sudo dnf install fuse-sshfs (Fedora)"
	default:
		return "SSHFS is not available for this platform"
	}
}

// NewSSHFSMount creates a new SSHFS mount
func NewSSHFSMount(host, remotePath, configFile string) (*SSHFSMount, error) {
	if !IsSSHFSAvailable() {
		return nil, fmt.Errorf("sshfs not installed. %s", GetSSHFSInstallInstructions())
	}

	// Create a temporary mount point
	mountPoint, err := os.MkdirTemp("", fmt.Sprintf("sshm-%s-", host))
	if err != nil {
		return nil, fmt.Errorf("failed to create mount point: %w", err)
	}

	return &SSHFSMount{
		Host:       host,
		RemotePath: remotePath,
		MountPoint: mountPoint,
		ConfigFile: configFile,
	}, nil
}

// Mount mounts the remote filesystem
func (m *SSHFSMount) Mount() error {
	// Build sshfs command
	// sshfs user@host:/path /mount/point -o options
	remote := fmt.Sprintf("%s:%s", m.Host, m.RemotePath)

	args := []string{remote, m.MountPoint}

	// Add SSH config file if specified
	if m.ConfigFile != "" {
		args = append(args, "-o", fmt.Sprintf("ssh_command=ssh -F %s", m.ConfigFile))
	}

	// Add useful options
	args = append(args,
		"-o", "reconnect",           // Auto-reconnect
		"-o", "ServerAliveInterval=15", // Keep connection alive
		"-o", "follow_symlinks",     // Follow symlinks
	)

	// On macOS, add volname for nicer display in Finder
	if runtime.GOOS == "darwin" {
		volName := fmt.Sprintf("%s:%s", m.Host, m.RemotePath)
		if len(volName) > 27 { // macOS volume name limit
			volName = m.Host
		}
		args = append(args, "-o", fmt.Sprintf("volname=%s", volName))
	}

	cmd := exec.Command("sshfs", args...)
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		// Clean up mount point on failure
		os.Remove(m.MountPoint)
		return fmt.Errorf("failed to mount: %w", err)
	}

	// Give it a moment to fully mount
	time.Sleep(500 * time.Millisecond)

	return nil
}

// Unmount unmounts the remote filesystem
func (m *SSHFSMount) Unmount() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// On macOS, use diskutil or umount
		cmd = exec.Command("diskutil", "unmount", "force", m.MountPoint)
	case "linux":
		cmd = exec.Command("fusermount", "-u", m.MountPoint)
	default:
		cmd = exec.Command("umount", m.MountPoint)
	}

	err := cmd.Run()

	// Try to remove the mount point directory
	os.Remove(m.MountPoint)

	return err
}

// ToRemotePath converts a local path within the mount to a remote path
func (m *SSHFSMount) ToRemotePath(localPath string) (string, error) {
	// Make sure the path is within the mount point
	absLocal, err := filepath.Abs(localPath)
	if err != nil {
		return "", err
	}

	absMountPoint, err := filepath.Abs(m.MountPoint)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absLocal, absMountPoint) {
		return "", fmt.Errorf("path is not within mount point")
	}

	// Get the relative path from mount point
	relPath, err := filepath.Rel(absMountPoint, absLocal)
	if err != nil {
		return "", err
	}

	// Combine with remote path
	if relPath == "." {
		return m.RemotePath, nil
	}

	return filepath.Join(m.RemotePath, relPath), nil
}

// OpenRemoteFilePicker mounts remote filesystem and opens native file picker
func OpenRemoteFilePicker(host, startPath, configFile string, mode PickerMode, title string) (*PickerResult, error) {
	if !IsSSHFSAvailable() {
		return nil, fmt.Errorf("sshfs not installed. %s", GetSSHFSInstallInstructions())
	}

	if !IsPickerAvailable() {
		return nil, fmt.Errorf("native file picker not available")
	}

	// Default to home directory if no start path
	if startPath == "" {
		startPath = "~"
	}

	// Create and mount
	mount, err := NewSSHFSMount(host, startPath, configFile)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Mounting %s:%s...\n", host, startPath)
	if err := mount.Mount(); err != nil {
		return nil, err
	}

	// Ensure we unmount when done
	defer func() {
		fmt.Println("Unmounting...")
		mount.Unmount()
	}()

	fmt.Println("Opening file picker...")

	// Open the native file picker pointing to the mount
	result, err := OpenFilePicker(mode, title, mount.MountPoint)
	if err != nil {
		return nil, err
	}

	if !result.Selected {
		return &PickerResult{Selected: false}, nil
	}

	// Convert the selected path back to remote path
	remotePath, err := mount.ToRemotePath(result.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path: %w", err)
	}

	return &PickerResult{
		Selected: true,
		Path:     remotePath,
	}, nil
}

// OpenRemoteFolderInFinder mounts and opens Finder, returns selected path when done
func OpenRemoteFolderInFinder(host, startPath, configFile string) (*PickerResult, error) {
	return OpenRemoteFilePicker(host, startPath, configFile, PickFile, "Select remote file")
}

// OpenRemoteDirectoryPicker opens picker for selecting remote directory
func OpenRemoteDirectoryPicker(host, startPath, configFile string) (*PickerResult, error) {
	return OpenRemoteFilePicker(host, startPath, configFile, PickDirectory, "Select remote folder")
}
