package cmd

import (
	"github.com/nais/frontend-plattform-console/pkg/config"
	"github.com/nais/frontend-plattform-console/pkg/server"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the server",
	Long:  `Run the server and start listening for requests`,
	Run: func(cmd *cobra.Command, args []string) {
		config := config.New()
		server.Run(config)
	},
}
