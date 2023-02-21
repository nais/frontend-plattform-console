package cmd

import (
	"github.com/nais/frontend-plattform-console/pkg/config"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "fpc",
		Short: "Frontend Platform Console",
		Long: `Frontend Platform Console is a web application that allows you to
		manage your frontend applications and their features.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	config.Setup(rootCmd)
}
