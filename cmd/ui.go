package cmd

import (
	"fmt"
	"os"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/spf13/cobra"
)

var (
	datacenter string
)

// uiCmd represents the ui command
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Opens up the Consul UI",
	Run: func(cmd *cobra.Command, args []string) {
		logger := createLogger()

		client := server.NewClient(socket)

		consul, err := client.GetConsul(datacenter)
		if err != nil {
			logger.Error("unable to fetch consul instance", "err", err)
			os.Exit(1)
		}

		port := consul.NamedPorts["http"]
		if port == 0 {
			logger.Error("consul HTTP port not registered")
			os.Exit(1)
		}

		url := fmt.Sprintf("http://127.0.0.1:%d", port)
		if err := open(url); err != nil {
			logger.Error("unable to open web page", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)

	uiCmd.Flags().StringVarP(&datacenter, "datacenter", "d", "dc1", "Open the Consul instance for the given datacenter.")
}
