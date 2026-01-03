package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
  2. Run the install script to update to the latest version

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

	fmt.Println("Running install script...")

	// Run the install script via curl and bash
	installCmd := exec.Command("bash", "-c", "curl -fsSL https://raw.githubusercontent.com/xvertile/sshc/main/install/install.sh | bash")
	installCmd.Stdin = os.Stdin
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running install script: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nUpdate complete!")
}

func init() {
	RootCmd.AddCommand(updateCmd)

	updateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force reinstall even if already on latest version")
}
