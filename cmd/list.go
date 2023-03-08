package cmd

import (
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/andrewstucki/consul-services/pkg/tables"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

var (
	allKinds bool
	kinds    []string

	defaultKinds = []string{
		"ingress",
		"service",
		"api",
	}
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists the services currently running.",
	Run: func(cmd *cobra.Command, args []string) {
		logger := hclog.Default()

		filteredKinds := []string{}
		if !allKinds {
			filteredKinds = kinds
		}

		client := server.NewClient(socket)
		services, err := client.List(filteredKinds...)
		if err != nil {
			logger.Error("unable to fetch services", "err", err)
			os.Exit(1)
		}

		tables.PrintServices(os.Stdout, services)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringSliceVarP(&kinds, "kind", "k", defaultKinds, "Specify kinds to filter by.")
	listCmd.Flags().BoolVarP(&allKinds, "all", "a", false, "List all service kinds.")
}
