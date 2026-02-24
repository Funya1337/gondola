package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tests defined in deploy.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := LoadConfig("deploy.yaml")
		if err != nil {
			return err
		}

		if cfg.Test.Skip {
			fmt.Println("Tests skipped (skip: true in deploy.yaml)")
			return nil
		}

		if len(cfg.Test.Commands) == 0 {
			return fmt.Errorf("no test commands defined in deploy.yaml")
		}

		for _, command := range cfg.Test.Commands {
			fmt.Printf("Running: %s\n", command)

			testCmd := exec.Command("bash", "-c", command)
			testCmd.Stdout = os.Stdout
			testCmd.Stderr = os.Stderr
			testCmd.Env = os.Environ()

			if err := testCmd.Run(); err != nil {
				return fmt.Errorf("test failed: %s: %w", command, err)
			}
		}

		fmt.Println("All tests passed!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
