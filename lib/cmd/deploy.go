package cmd

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

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
		if dc.Port == 0 {
			dc.Port = 22
		}

		keyPath := expandHome(dc.KeyPath)
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH key %s: %w", keyPath, err)
		}

		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse SSH key: %w", err)
		}

		sshConfig := &ssh.ClientConfig{
			User: dc.User,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		addr := fmt.Sprintf("%s:%d", dc.Host, dc.Port)
		fmt.Printf("Connecting to %s@%s...\n", dc.User, addr)

		client, err := ssh.Dial("tcp", addr, sshConfig)
		if err != nil {
			return fmt.Errorf("SSH connection failed: %w", err)
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

func runRemoteCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	fmt.Printf("Running: %s\n", command)
	return session.Run(command)
}

func uploadFile(client *ssh.Client, localPath, remotePath string) error {
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP session failed: %w", err)
	}
	defer sftpClient.Close()

	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file %s: %w", localPath, err)
	}
	defer local.Close()

	remote, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer remote.Close()

	_, err = io.Copy(remote, local)
	return err
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func init() {
	rootCmd.AddCommand(deployCmd)
}
