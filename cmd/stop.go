package cmd

import (
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops a daemonized run",
	Run: func(cmd *cobra.Command, args []string) {
		logger := createLogger()
		client := server.NewClient(socket)
		if err := client.Shutdown(); err != nil {
			logger.Error("unable to shut server down", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
