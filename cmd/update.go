package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/xvertile/sshc/internal/version"

	"github.com/spf13/cobra"
)

var (
	forceUpdate bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sshc to the latest version",
	Long: `Check for and install the latest version of sshc.

This command will:
  1. Check GitHub for the latest release
  2. Download the appropriate binary for your system
  3. Replace the current binary with the new version

Examples:
  sshc update         # Update to latest version if available
  sshc update --force # Force reinstall even if already on latest`,
	Run: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) {
	fmt.Println("Checking for updates...")

	// Check for updates
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updateInfo, err := version.CheckForUpdates(ctx, AppVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if AppVersion == "dev" {
		fmt.Println("Running development version - cannot auto-update.")
		fmt.Println("Please build from source or install a release version.")
		os.Exit(0)
	}

	if !updateInfo.Available && !forceUpdate {
		fmt.Printf("Already on latest version (%s)\n", AppVersion)
		os.Exit(0)
	}

	if updateInfo.Available {
		fmt.Printf("Update available: %s -> %s\n", updateInfo.CurrentVer, updateInfo.LatestVer)
	} else {
		fmt.Printf("Reinstalling current version (%s)\n", AppVersion)
	}

	// Determine OS and architecture
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	if goos != "darwin" && goos != "linux" {
		fmt.Fprintf(os.Stderr, "Unsupported operating system: %s\n", goos)
		os.Exit(1)
	}

	if goarch != "amd64" && goarch != "arm64" {
		fmt.Fprintf(os.Stderr, "Unsupported architecture: %s\n", goarch)
		os.Exit(1)
	}

	// Get the version to download
	downloadVersion := updateInfo.LatestVer
	if !updateInfo.Available && forceUpdate {
		downloadVersion = AppVersion
	}

	// Strip 'v' prefix for filename
	versionNum := strings.TrimPrefix(downloadVersion, "v")

	// Construct download URL
	filename := fmt.Sprintf("sshc_%s_%s_%s.tar.gz", versionNum, goos, goarch)
	downloadURL := fmt.Sprintf("https://github.com/xvertile/sshc/releases/download/%s/%s", downloadVersion, filename)

	fmt.Printf("Downloading %s...\n", filename)

	// Download the release
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading update: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error downloading update: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "sshc-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	// Extract the tarball
	fmt.Println("Extracting...")
	if err := extractTarGz(resp.Body, tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting update: %v\n", err)
		os.Exit(1)
	}

	// Find the binary
	newBinary := filepath.Join(tmpDir, "sshc")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: binary not found in archive\n")
		os.Exit(1)
	}

	// Get current binary path
	currentBinary, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding current binary: %v\n", err)
		os.Exit(1)
	}

	// Resolve symlinks
	currentBinary, err = filepath.EvalSymlinks(currentBinary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving binary path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Installing to %s...\n", currentBinary)

	// Check if we need sudo
	needSudo := false
	if err := os.Rename(newBinary, currentBinary); err != nil {
		if os.IsPermission(err) {
			needSudo = true
		} else {
			// Try copying instead (cross-device link)
			if err := copyFile(newBinary, currentBinary); err != nil {
				if os.IsPermission(err) {
					needSudo = true
				} else {
					fmt.Fprintf(os.Stderr, "Error installing update: %v\n", err)
					os.Exit(1)
				}
			}
		}
	}

	if needSudo {
		fmt.Println("Need elevated privileges to install...")
		sudoCmd := exec.Command("sudo", "cp", newBinary, currentBinary)
		sudoCmd.Stdin = os.Stdin
		sudoCmd.Stdout = os.Stdout
		sudoCmd.Stderr = os.Stderr
		if err := sudoCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error installing with sudo: %v\n", err)
			os.Exit(1)
		}

		// Set permissions
		chmodCmd := exec.Command("sudo", "chmod", "+x", currentBinary)
		if err := chmodCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting permissions: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Set permissions
		if err := os.Chmod(currentBinary, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting permissions: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("\nSuccessfully updated to %s\n", downloadVersion)

	// Show new version
	versionCmd := exec.Command(currentBinary, "--version")
	versionCmd.Stdout = os.Stdout
	versionCmd.Stderr = os.Stderr
	versionCmd.Run()
}

// extractTarGz extracts a tar.gz archive to the destination directory
func extractTarGz(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func init() {
	RootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force reinstall even if already on latest version")
}
