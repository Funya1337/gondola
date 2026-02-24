package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the Go application using settings from deploy.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		buildArgs := []string{"build"}

		if cfg.Build.LDFlags != "" {
			buildArgs = append(buildArgs, "-ldflags", cfg.Build.LDFlags)
		}

		if cfg.Build.Output != "" {
			buildArgs = append(buildArgs, "-o", cfg.Build.Output)
		}

		if cfg.Build.Entry != "" {
			buildArgs = append(buildArgs, cfg.Build.Entry)
		}

		goCmd := exec.Command("go", buildArgs...)
		goCmd.Stdout = os.Stdout
		goCmd.Stderr = os.Stderr
		goCmd.Env = os.Environ()

		if cfg.Build.GOOS != "" {
			goCmd.Env = append(goCmd.Env, "GOOS="+cfg.Build.GOOS)
		}
		if cfg.Build.GOARCH != "" {
			goCmd.Env = append(goCmd.Env, "GOARCH="+cfg.Build.GOARCH)
		}
		for _, env := range cfg.Build.ExtraEnv {
			goCmd.Env = append(goCmd.Env, env)
		}

		fmt.Printf("Building %s (%s/%s)...\n", cfg.Project.Name, cfg.Build.GOOS, cfg.Build.GOARCH)

		if err := goCmd.Run(); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		fmt.Printf("Built successfully: %s\n", cfg.Build.Output)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
}
