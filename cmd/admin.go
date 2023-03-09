package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/spf13/cobra"
)

// adminCmd represents the admin command
var adminCmd = &cobra.Command{
	Use:   "admin [name]",
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Short: "Opens the envoy admin panel for a given service.",
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		logger := createLogger()

		client := server.NewClient(socket)
		service, err := client.Get(kind, name)
		if err != nil {
			logger.Error("unable to fetch service", "err", err)
			os.Exit(1)
		}

		if service.AdminPort == 0 {
			logger.Error("service is not a proxy")
			os.Exit(1)
		}

		url := fmt.Sprintf("http://127.0.0.1:%d", service.AdminPort)
		if err := open(url); err != nil {
			logger.Error("unable to open web page", "err", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(adminCmd)

	adminCmd.Flags().StringVarP(&kind, "kind", "k", defaultKind, "Kind of service to lookup.")
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}
