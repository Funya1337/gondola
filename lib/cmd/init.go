package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const defaultConfig = `# go-deploy configuration

project:
  name: "myapp"

build:
  entry: "."
  output: "bin/myapp"
  goos: "linux"
  goarch: "amd64"
  ldflags: "-s -w"
  extra_env:
    - "CGO_ENABLED=0"

test:
  commands:
    - "go test ./..."
  skip: false

deploy:
  host: ""
  port: 22
  user: ""
  key_path: "~/.ssh/id_rsa"
  remote_path: "/opt/myapp/myapp"
  pre_deploy:
    - "systemctl stop myapp"
  post_deploy:
    - "chmod +x /opt/myapp/myapp"
    - "systemctl start myapp"
`

var forceOverwrite bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a sample deploy.yaml configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		const filename = "deploy.yaml"

		if _, err := os.Stat(filename); err == nil && !forceOverwrite {
			return fmt.Errorf("%s already exists (use --force to overwrite)", filename)
		}

		if err := os.WriteFile(filename, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		fmt.Printf("Created %s\n", filename)
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&forceOverwrite, "force", false, "overwrite existing deploy.yaml")
	rootCmd.AddCommand(initCmd)
}
