package cmd

import (
	"github.com/nais/bifrost/pkg/config"
	"github.com/nais/bifrost/pkg/server"
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
		config := config.New(cmd.Context())
		server.Run(config)
	},
}
