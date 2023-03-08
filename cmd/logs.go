package cmd

import (
	"io"
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Read logs from a deployed service.",
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		logger := hclog.Default()

		client := server.NewClient(socket)
		service, err := client.Get(kind, name)
		if err != nil {
			logger.Error("unable to fetch service", "err", err)
			os.Exit(1)
		}
		if service.Logs != "" {
			logFile, err := os.Open(service.Logs)
			if err != nil {
				logger.Error("unable to open logs", "err", err)
				os.Exit(1)
			}
			_, err = io.Copy(os.Stdout, logFile)
			if err != nil {
				logger.Error("unable to read logs", "err", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().StringVarP(&kind, "kind", "k", "connect-proxy", "Kind of service to lookup.")
}
