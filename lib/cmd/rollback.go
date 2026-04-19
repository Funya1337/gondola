package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback to the previous version from backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		dc := cfg.Deploy
		if err := validateDeploySSH(dc); err != nil {
			return err
		}
		if dc.RemotePath == "" {
			return fmt.Errorf("deploy.remote_path is required in deploy.yaml")
		}

		client, err := sshConnect(dc)
		if err != nil {
			return err
		}
		defer client.Close()

		backupPath := dc.RemotePath + ".bak"

		// Check backup exists
		out, err := runRemoteCommandOutput(client, fmt.Sprintf("test -f %s && echo exists", backupPath))
		if err != nil || !strings.Contains(out, "exists") {
			return fmt.Errorf("no backup found at %s — nothing to rollback to", backupPath)
		}

		svcName := dc.Service.Name

		// Stop the service
		if svcName != "" {
			fmt.Printf("Stopping service %s...\n", svcName)
			_ = runRemoteCommand(client, fmt.Sprintf("sudo systemctl stop %s", svcName))
		}

		// Restore backup
		fmt.Printf("Restoring %s -> %s...\n", backupPath, dc.RemotePath)
		if err := runRemoteCommand(client, fmt.Sprintf("cp -f %s %s", backupPath, dc.RemotePath)); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}

		// Make executable
		if err := runRemoteCommand(client, fmt.Sprintf("chmod +x %s", dc.RemotePath)); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}

		// Restart the service
		if svcName != "" {
			fmt.Printf("Starting service %s...\n", svcName)
			if err := runRemoteCommand(client, fmt.Sprintf("sudo systemctl start %s", svcName)); err != nil {
				return fmt.Errorf("failed to start service after rollback: %w", err)
			}

			out, _ := runRemoteCommandOutput(client, fmt.Sprintf("systemctl is-active %s", svcName))
			fmt.Printf("Service status: %s", out)
		}

		fmt.Println("Rollback complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}
