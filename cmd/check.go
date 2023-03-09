/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check [name] [upstream]",
	Short: "Checks for one-way connectivity between two services",
	Args:  cobra.MatchAll(cobra.ExactArgs(2)),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		upstream := args[1]
		logger := createLogger()

		// normalize the name to add the -proxy so that we know we're querying
		// the upstream port from the connect proxy rather than from the service
		// itself
		if !strings.HasSuffix(name, "-proxy") {
			name += "-proxy"
		}

		client := server.NewClient(socket)
		service, err := client.Get(kind, name)
		if err != nil {
			logger.Error("unable to fetch service", "err", err)
			os.Exit(1)
		}

		port, ok := service.NamedPorts[upstream]
		if !ok {
			logger.Error("service does not have upstream defined", "upstream", upstream)
			os.Exit(1)
		}

		response, err := checkConnectivity(port)
		if err != nil {
			logger.Error("error connecting to upstream", "err", err)
			os.Exit(1)
		}

		logger.Info("response received", "response", response)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)

	checkCmd.Flags().StringVarP(&kind, "kind", "k", defaultKind, "Kind of service to lookup.")
}

func checkConnectivity(port int) (string, error) {
	url := fmt.Sprintf("http://localhost:%d", port)
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
