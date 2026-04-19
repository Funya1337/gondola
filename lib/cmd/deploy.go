package cmd

import (
	"fmt"
	"io"
	"path"

	"github.com/pkg/sftp"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the built binary to a remote server via SSH",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		dc := cfg.Deploy
		if dc.Host == "" {
			return fmt.Errorf("deploy.host is required in deploy.yaml")
		}
		if dc.User == "" {
			return fmt.Errorf("deploy.user is required in deploy.yaml")
		}
		if dc.RemotePath == "" {
			return fmt.Errorf("deploy.remote_path is required in deploy.yaml")
		}

		client, err := sshConnect(dc)
		if err != nil {
			return err
		}
		defer client.Close()

		fmt.Println("Connected.")

		// Run pre-deploy commands
		for _, command := range dc.PreDeploy {
			if err := runRemoteCommand(client, command); err != nil {
				return fmt.Errorf("pre_deploy command failed (%s): %w", command, err)
			}
		}

		// Upload binary via SFTP
		localPath := cfg.Build.Output
		if localPath == "" {
			return fmt.Errorf("build.output is required in deploy.yaml")
		}

		// Ensure remote directory exists
		remoteDir := path.Dir(dc.RemotePath)
		if err := runRemoteCommand(client, fmt.Sprintf("sudo mkdir -p %s && sudo chown %s %s", remoteDir, dc.User, remoteDir)); err != nil {
			return fmt.Errorf("failed to create remote directory %s: %w", remoteDir, err)
		}

		// Stop the service if running (required to replace the binary)
		if dc.Service.Name != "" {
			fmt.Printf("Stopping service %s before deploy...\n", dc.Service.Name)
			_, _ = runRemoteCommandOutput(client, fmt.Sprintf("sudo systemctl stop %s 2>/dev/null", dc.Service.Name))
		}

		// Backup current binary before overwriting
		backupPath := dc.RemotePath + ".bak"
		fmt.Printf("Backing up current binary to %s...\n", backupPath)
		// Ignore error — file may not exist on first deploy
		_, _ = runRemoteCommandOutput(client, fmt.Sprintf("cp -f %s %s 2>/dev/null", dc.RemotePath, backupPath))

		// Remove old binary so SFTP can create a fresh file (avoids "text file busy")
		_, _ = runRemoteCommandOutput(client, fmt.Sprintf("rm -f %s", dc.RemotePath))

		fmt.Printf("Uploading %s -> %s...\n", localPath, dc.RemotePath)
		if err := uploadFile(client, localPath, dc.RemotePath); err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}
		fmt.Println("Upload complete.")

		// Install systemd service file if configured
		if dc.Service.Name != "" {
			unitContent := fmt.Sprintf("[Unit]\nDescription=%s\nAfter=network.target\n\n[Service]\nType=simple\nExecStart=%s\nRestart=%s\n\n[Install]\nWantedBy=multi-user.target\n",
				dc.Service.Description,
				dc.RemotePath,
				dc.Service.Restart,
			)

			fmt.Printf("Installing systemd service %s...\n", dc.Service.Name)
			installCmd := fmt.Sprintf("sudo tee /etc/systemd/system/%s.service > /dev/null << 'SERVICEEOF'\n%sSERVICEEOF", dc.Service.Name, unitContent)
			if err := runRemoteCommand(client, installCmd); err != nil {
				return fmt.Errorf("failed to install service file: %w", err)
			}

			if err := runRemoteCommand(client, "sudo systemctl daemon-reload"); err != nil {
				return fmt.Errorf("failed to reload systemd: %w", err)
			}

			enableCmd := fmt.Sprintf("sudo systemctl enable %s", dc.Service.Name)
			if err := runRemoteCommand(client, enableCmd); err != nil {
				return fmt.Errorf("failed to enable service: %w", err)
			}
		}

		// Run post-deploy commands
		for _, command := range dc.PostDeploy {
			if err := runRemoteCommand(client, command); err != nil {
				return fmt.Errorf("post_deploy command failed (%s): %w", command, err)
			}
		}

		fmt.Println("Deploy complete.")
		return nil
	},
}

func newSFTPClient(client *ssh.Client) (*sftp.Client, error) {
	c, err := sftp.NewClient(client)
	if err != nil {
		return nil, fmt.Errorf("SFTP session failed: %w", err)
	}
	return c, nil
}

func copyIO(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
