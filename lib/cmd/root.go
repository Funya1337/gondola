package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-deploy",
	Short: "Deploy Go applications to Linux servers via SSH",
}

func Execute() error {
	return rootCmd.Execute()
}
