package cmd

import (
	"github.com/nais/frontend-plattform-console/pkg/config"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "bifrost",
		Short: "Bifrost Frontend Platform Portal",
		Long: `Frontend Platform Console is a web portal that allows developer to
		manage resources for frontend applications.`,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	config.Setup(rootCmd)
}
