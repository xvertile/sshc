package cmd

import (
	"fmt"
	"os"

	"github.com/Gu1llaum-3/sshm/internal/config"
	"github.com/Gu1llaum-3/sshm/internal/history"
	"github.com/Gu1llaum-3/sshm/internal/transfer"
	"github.com/Gu1llaum-3/sshm/internal/ui"

	"github.com/spf13/cobra"
)

var (
	cpRecursive bool
)

var cpCmd = &cobra.Command{
	Use:   "cp <source> <destination>",
	Short: "Copy files to/from SSH hosts",
	Long: `Copy files to or from SSH hosts using SCP.

The source or destination should be in the format host:/path for remote paths.
Local paths can be relative or absolute.

Examples:
  # Upload a file
  sshm cp ./local-file.txt myhost:/remote/path/

  # Download a file
  sshm cp myhost:/var/log/app.log ./downloads/

  # Upload a directory (recursive)
  sshm cp -r ./my-folder myhost:/remote/path/

  # Interactive mode (opens transfer UI)
  sshm cp myhost`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If only one argument (host), open interactive transfer UI
		if len(args) == 1 {
			hostName := args[0]
			return runInteractiveTransfer(hostName)
		}

		// Two arguments: source and destination
		source := args[0]
		dest := args[1]

		// Parse the transfer request
		req, err := transfer.ParseTransferArgs(source, dest)
		if err != nil {
			return err
		}

		// Override recursive flag if explicitly set
		if cpRecursive {
			req.Recursive = true
		}

		// Set config file if specified
		req.ConfigFile = configFile

		// Verify the host exists in SSH config
		var hostExists bool
		if configFile != "" {
			hostExists, err = config.QuickHostExistsInFile(req.Host, configFile)
		} else {
			hostExists, err = config.QuickHostExists(req.Host)
		}

		if err != nil {
			return fmt.Errorf("error checking SSH config: %w", err)
		}

		if !hostExists {
			return fmt.Errorf("host '%s' not found in SSH configuration", req.Host)
		}

		// Execute the transfer
		direction := "upload"
		if req.Direction == transfer.Download {
			direction = "download"
		}

		fmt.Printf("Transferring %s %s...\n", direction, req.LocalPath)

		result := req.ExecuteWithProgress()
		if !result.Success {
			return fmt.Errorf("transfer failed: %w", result.Error)
		}

		// Record the transfer in history
		historyManager, err := history.NewHistoryManager()
		if err == nil {
			_ = historyManager.RecordTransfer(req.Host, direction, req.LocalPath, req.RemotePath)
		}

		fmt.Println("Transfer complete!")
		return nil
	},
}

func runInteractiveTransfer(hostName string) error {
	// Verify the host exists
	var hostExists bool
	var err error

	if configFile != "" {
		hostExists, err = config.QuickHostExistsInFile(hostName, configFile)
	} else {
		hostExists, err = config.QuickHostExists(hostName)
	}

	if err != nil {
		return fmt.Errorf("error checking SSH config: %w", err)
	}

	if !hostExists {
		return fmt.Errorf("host '%s' not found in SSH configuration", hostName)
	}

	// Run the transfer TUI
	return ui.RunTransferForm(hostName, configFile)
}

func init() {
	RootCmd.AddCommand(cpCmd)

	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "Copy directories recursively")
}

var sendCmd = &cobra.Command{
	Use:   "send <host> [local-path]",
	Short: "Upload files to an SSH host",
	Long: `Upload files to an SSH host. Opens a native file picker if no path is specified.

Examples:
  # Upload with native file picker
  sshm send myhost

  # Upload a specific file
  sshm send myhost ./file.txt`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostName := args[0]

		// Verify the host exists
		var hostExists bool
		var err error

		if configFile != "" {
			hostExists, err = config.QuickHostExistsInFile(hostName, configFile)
		} else {
			hostExists, err = config.QuickHostExists(hostName)
		}

		if err != nil {
			return fmt.Errorf("error checking SSH config: %w", err)
		}

		if !hostExists {
			return fmt.Errorf("host '%s' not found in SSH configuration", hostName)
		}

		var localPath string

		if len(args) == 1 {
			// No path given - try native file picker first
			if transfer.IsPickerAvailable() {
				cwd, _ := os.Getwd()
				result, err := transfer.OpenFilePicker(transfer.PickFile, "Select file to upload", cwd)
				if err != nil {
					return fmt.Errorf("file picker error: %w", err)
				}
				if !result.Selected {
					fmt.Println("No file selected, cancelled.")
					return nil
				}
				localPath = result.Path
			} else {
				// Fall back to TUI
				return ui.RunTransferFormWithDirection(hostName, configFile, transfer.Upload)
			}
		} else {
			localPath = args[1]
		}

		// Expand and validate the path
		expandedPath, err := transfer.ExpandPath(localPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		if err := transfer.ValidateLocalPath(expandedPath, transfer.Upload); err != nil {
			return err
		}

		// Get remote destination - use TUI browser
		var remotePath string
		path, selected, err := ui.RunRemoteBrowser(hostName, "~", configFile, ui.BrowseDirectories)
		if err != nil {
			fmt.Printf("Remote browser error: %v\n", err)
			fmt.Print("Remote destination path (default ~/): ")
			fmt.Scanln(&remotePath)
		} else if !selected {
			fmt.Println("No destination selected, cancelled.")
			return nil
		} else {
			remotePath = path
		}

		if remotePath == "" {
			remotePath = "~/"
		}

		req := &transfer.TransferRequest{
			Host:       hostName,
			Direction:  transfer.Upload,
			LocalPath:  expandedPath,
			RemotePath: remotePath,
			ConfigFile: configFile,
		}

		// Check if it's a directory
		info, _ := os.Stat(expandedPath)
		if info != nil && info.IsDir() {
			req.Recursive = true
		}

		fmt.Printf("Uploading %s to %s:%s...\n", localPath, hostName, remotePath)
		result := req.ExecuteWithProgress()

		if !result.Success {
			return fmt.Errorf("upload failed: %w", result.Error)
		}

		// Record in history
		historyManager, err := history.NewHistoryManager()
		if err == nil {
			_ = historyManager.RecordTransfer(hostName, "upload", expandedPath, remotePath)
		}

		fmt.Println("Upload complete!")
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <host> [remote-path]",
	Short: "Download files from an SSH host",
	Long: `Download files from an SSH host. Opens file browsers for selection.

Examples:
  # Browse remote files to download (opens TUI file browser)
  sshm get myhost

  # Download a specific file (opens local folder picker for destination)
  sshm get myhost /var/log/app.log

  # Download to specific location (no pickers)
  sshm get myhost /var/log/app.log ./downloads/`,
	Args: cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		hostName := args[0]

		// Verify the host exists
		var hostExists bool
		var err error

		if configFile != "" {
			hostExists, err = config.QuickHostExistsInFile(hostName, configFile)
		} else {
			hostExists, err = config.QuickHostExists(hostName)
		}

		if err != nil {
			return fmt.Errorf("error checking SSH config: %w", err)
		}

		if !hostExists {
			return fmt.Errorf("host '%s' not found in SSH configuration", hostName)
		}

		var remotePath string
		var localPath string

		// Handle remote path
		if len(args) >= 2 {
			remotePath = args[1]
		} else {
			// No remote path - use TUI browser
			path, selected, err := ui.RunRemoteBrowser(hostName, "~", configFile, ui.BrowseFiles)
			if err != nil {
				return fmt.Errorf("remote browser error: %w", err)
			}
			if !selected {
				fmt.Println("No file selected, cancelled.")
				return nil
			}
			remotePath = path
		}

		// Handle local path
		if len(args) >= 3 {
			localPath = args[2]
		} else {
			// No local path given - try native folder picker
			if transfer.IsPickerAvailable() {
				cwd, _ := os.Getwd()
				result, err := transfer.OpenFilePicker(transfer.PickDirectory, "Select download destination", cwd)
				if err != nil {
					return fmt.Errorf("file picker error: %w", err)
				}
				if !result.Selected {
					fmt.Println("No destination selected, cancelled.")
					return nil
				}
				localPath = result.Path
			} else {
				// Fall back to asking
				fmt.Print("Local destination path (default: ./): ")
				fmt.Scanln(&localPath)
				if localPath == "" {
					localPath = "./"
				}
			}
		}

		// Expand the local path
		expandedPath, err := transfer.ExpandPath(localPath)
		if err != nil {
			return fmt.Errorf("invalid path: %w", err)
		}

		req := &transfer.TransferRequest{
			Host:       hostName,
			Direction:  transfer.Download,
			LocalPath:  expandedPath,
			RemotePath: remotePath,
			ConfigFile: configFile,
		}

		fmt.Printf("Downloading %s:%s to %s...\n", hostName, remotePath, localPath)
		result := req.ExecuteWithProgress()

		if !result.Success {
			return fmt.Errorf("download failed: %w", result.Error)
		}

		// Record in history
		historyManager, err := history.NewHistoryManager()
		if err == nil {
			_ = historyManager.RecordTransfer(hostName, "download", expandedPath, remotePath)
		}

		fmt.Println("Download complete!")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(sendCmd)
	RootCmd.AddCommand(getCmd)
}
