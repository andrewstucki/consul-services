package cmd

import (
	"io"
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/andrewstucki/consul-services/pkg/tables"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/jsonpath"
)

var (
	kind   string
	filter string

	defaultKind = "service"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Gets a particular service",
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

		if err := printResults(os.Stdout, service, filter); err != nil {
			logger.Error("unable to filter service", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)

	getCmd.Flags().StringVarP(&kind, "kind", "k", defaultKind, "Kind of service to lookup.")
	getCmd.Flags().StringVarP(&filter, "filter", "f", "", "Filter the results.")
}

func printResults(w io.Writer, service *server.Service, filterExpr string) error {
	if filterExpr == "" {
		tables.PrintServices(w, []server.Service{*service})
		return nil
	}

	filterExpr, err := relaxedJSONPathExpression(filterExpr)
	if err != nil {
		return err
	}

	filter := jsonpath.New("filter")
	if err := filter.Parse(filterExpr); err != nil {
		return err
	}

	results, err := filter.FindResults(service)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}

	for _, result := range results {
		if err := filter.PrintResults(w, result); err != nil {
			return err
		}
	}

	return nil
}
