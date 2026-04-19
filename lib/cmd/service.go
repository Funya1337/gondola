package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the remote service via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		dc := cfg.Deploy
		if err := validateDeploySSH(dc); err != nil {
			return err
		}

		client, err := sshConnect(dc)
		if err != nil {
			return err
		}
		defer client.Close()

		svcName := dc.Service.Name
		fmt.Printf("Stopping service %s on %s...\n", svcName, dc.Host)

		if err := runRemoteCommand(client, fmt.Sprintf("sudo systemctl stop %s", svcName)); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}

		fmt.Println("Service stopped.")
		return nil
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the remote service via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		dc := cfg.Deploy
		if err := validateDeploySSH(dc); err != nil {
			return err
		}

		client, err := sshConnect(dc)
		if err != nil {
			return err
		}
		defer client.Close()

		svcName := dc.Service.Name
		fmt.Printf("Restarting service %s on %s...\n", svcName, dc.Host)

		if err := runRemoteCommand(client, fmt.Sprintf("sudo systemctl restart %s", svcName)); err != nil {
			return fmt.Errorf("failed to restart service: %w", err)
		}

		// Check status after restart
		out, _ := runRemoteCommandOutput(client, fmt.Sprintf("systemctl is-active %s", svcName))
		fmt.Printf("Service status: %s", out)
		return nil
	},
}

func validateDeploySSH(dc DeployConfig) error {
	if dc.Host == "" {
		return fmt.Errorf("deploy.host is required in deploy.yaml")
	}
	if dc.User == "" {
		return fmt.Errorf("deploy.user is required in deploy.yaml")
	}
	if dc.Service.Name == "" {
		return fmt.Errorf("deploy.service.name is required in deploy.yaml")
	}
	return nil
}

func init() {
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
}
