package cmd

import (
	"fmt"
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/spf13/cobra"
)

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generates a shell script for a Github report",
	Run: func(cmd *cobra.Command, args []string) {
		logger := createLogger()

		client := server.NewClient(socket)
		report, err := client.GetReport()
		if err != nil {
			logger.Error("unable to fetch report", "err", err)
			os.Exit(1)
		}
		fmt.Println(report)
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)
}
