package transfer

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// PickerMode defines whether we're selecting files or directories
type PickerMode int

const (
	PickFile PickerMode = iota
	PickDirectory
	PickMultiple // Multiple files
)

// PickerResult contains the result of a file picker operation
type PickerResult struct {
	Selected  bool     // True if user selected something (didn't cancel)
	Path      string   // Single path (for PickFile/PickDirectory)
	Paths     []string // Multiple paths (for PickMultiple)
	Directory string   // The directory where selection was made
}

// OpenFilePicker opens the native OS file picker dialog
func OpenFilePicker(mode PickerMode, title string, startDir string) (*PickerResult, error) {
	switch runtime.GOOS {
	case "darwin":
		return openMacOSPicker(mode, title, startDir)
	case "linux":
		return openLinuxPicker(mode, title, startDir)
	default:
		return nil, fmt.Errorf("native file picker not supported on %s", runtime.GOOS)
	}
}

// OpenSavePicker opens a native save dialog to select destination
func OpenSavePicker(title string, defaultName string, startDir string) (*PickerResult, error) {
	switch runtime.GOOS {
	case "darwin":
		return openMacOSSavePicker(title, defaultName, startDir)
	case "linux":
		return openLinuxSavePicker(title, defaultName, startDir)
	default:
		return nil, fmt.Errorf("native file picker not supported on %s", runtime.GOOS)
	}
}

// IsPickerAvailable checks if native file picker is available on this system
func IsPickerAvailable() bool {
	switch runtime.GOOS {
	case "darwin":
		// osascript is always available on macOS
		_, err := exec.LookPath("osascript")
		return err == nil
	case "linux":
		// Check for zenity or kdialog
		if _, err := exec.LookPath("zenity"); err == nil {
			return true
		}
		if _, err := exec.LookPath("kdialog"); err == nil {
			return true
		}
		return false
	default:
		return false
	}
}

// macOS implementation using osascript
func openMacOSPicker(mode PickerMode, title string, startDir string) (*PickerResult, error) {
	var script string

	switch mode {
	case PickFile:
		script = fmt.Sprintf(`
			set defaultPath to POSIX file "%s"
			try
				set selectedFile to choose file with prompt "%s" default location defaultPath
				return POSIX path of selectedFile
			on error
				return ""
			end try
		`, escapeAppleScript(startDir), escapeAppleScript(title))

	case PickDirectory:
		script = fmt.Sprintf(`
			set defaultPath to POSIX file "%s"
			try
				set selectedFolder to choose folder with prompt "%s" default location defaultPath
				return POSIX path of selectedFolder
			on error
				return ""
			end try
		`, escapeAppleScript(startDir), escapeAppleScript(title))

	case PickMultiple:
		script = fmt.Sprintf(`
			set defaultPath to POSIX file "%s"
			try
				set selectedFiles to choose file with prompt "%s" default location defaultPath with multiple selections allowed
				set filePaths to ""
				repeat with f in selectedFiles
					set filePaths to filePaths & POSIX path of f & linefeed
				end repeat
				return filePaths
			on error
				return ""
			end try
		`, escapeAppleScript(startDir), escapeAppleScript(title))
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		// User cancelled or error occurred
		return &PickerResult{Selected: false}, nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return &PickerResult{Selected: false}, nil
	}

	if mode == PickMultiple {
		paths := strings.Split(result, "\n")
		var cleanPaths []string
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p != "" {
				cleanPaths = append(cleanPaths, p)
			}
		}
		return &PickerResult{
			Selected: true,
			Paths:    cleanPaths,
			Path:     cleanPaths[0],
		}, nil
	}

	return &PickerResult{
		Selected: true,
		Path:     result,
	}, nil
}

func openMacOSSavePicker(title string, defaultName string, startDir string) (*PickerResult, error) {
	script := fmt.Sprintf(`
		set defaultPath to POSIX file "%s"
		try
			set savePath to choose file name with prompt "%s" default name "%s" default location defaultPath
			return POSIX path of savePath
		on error
			return ""
		end try
	`, escapeAppleScript(startDir), escapeAppleScript(title), escapeAppleScript(defaultName))

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return &PickerResult{Selected: false}, nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return &PickerResult{Selected: false}, nil
	}

	return &PickerResult{
		Selected: true,
		Path:     result,
	}, nil
}

// Linux implementation using zenity or kdialog
func openLinuxPicker(mode PickerMode, title string, startDir string) (*PickerResult, error) {
	// Try zenity first, then kdialog
	if zenityPath, err := exec.LookPath("zenity"); err == nil {
		return openZenityPicker(zenityPath, mode, title, startDir)
	}

	if kdialogPath, err := exec.LookPath("kdialog"); err == nil {
		return openKdialogPicker(kdialogPath, mode, title, startDir)
	}

	return nil, fmt.Errorf("no file picker available (install zenity or kdialog)")
}

func openZenityPicker(zenityPath string, mode PickerMode, title string, startDir string) (*PickerResult, error) {
	args := []string{"--file-selection", "--title", title}

	if startDir != "" {
		args = append(args, "--filename", startDir+"/")
	}

	switch mode {
	case PickDirectory:
		args = append(args, "--directory")
	case PickMultiple:
		args = append(args, "--multiple", "--separator", "\n")
	}

	cmd := exec.Command(zenityPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// User cancelled
		return &PickerResult{Selected: false}, nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return &PickerResult{Selected: false}, nil
	}

	if mode == PickMultiple {
		paths := strings.Split(result, "\n")
		return &PickerResult{
			Selected: true,
			Paths:    paths,
			Path:     paths[0],
		}, nil
	}

	return &PickerResult{
		Selected: true,
		Path:     result,
	}, nil
}

func openKdialogPicker(kdialogPath string, mode PickerMode, title string, startDir string) (*PickerResult, error) {
	var args []string

	switch mode {
	case PickFile:
		args = []string{"--getopenfilename", startDir, "*", "--title", title}
	case PickDirectory:
		args = []string{"--getexistingdirectory", startDir, "--title", title}
	case PickMultiple:
		args = []string{"--getopenfilename", startDir, "*", "--multiple", "--separate-output", "--title", title}
	}

	cmd := exec.Command(kdialogPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return &PickerResult{Selected: false}, nil
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return &PickerResult{Selected: false}, nil
	}

	if mode == PickMultiple {
		paths := strings.Split(result, "\n")
		return &PickerResult{
			Selected: true,
			Paths:    paths,
			Path:     paths[0],
		}, nil
	}

	return &PickerResult{
		Selected: true,
		Path:     result,
	}, nil
}

func openLinuxSavePicker(title string, defaultName string, startDir string) (*PickerResult, error) {
	if zenityPath, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--save", "--confirm-overwrite", "--title", title}
		if startDir != "" {
			args = append(args, "--filename", startDir+"/"+defaultName)
		}

		cmd := exec.Command(zenityPath, args...)
		output, err := cmd.Output()
		if err != nil {
			return &PickerResult{Selected: false}, nil
		}

		result := strings.TrimSpace(string(output))
		if result == "" {
			return &PickerResult{Selected: false}, nil
		}

		return &PickerResult{Selected: true, Path: result}, nil
	}

	if kdialogPath, err := exec.LookPath("kdialog"); err == nil {
		args := []string{"--getsavefilename", startDir + "/" + defaultName, "*", "--title", title}

		cmd := exec.Command(kdialogPath, args...)
		output, err := cmd.Output()
		if err != nil {
			return &PickerResult{Selected: false}, nil
		}

		result := strings.TrimSpace(string(output))
		if result == "" {
			return &PickerResult{Selected: false}, nil
		}

		return &PickerResult{Selected: true, Path: result}, nil
	}

	return nil, fmt.Errorf("no file picker available (install zenity or kdialog)")
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
