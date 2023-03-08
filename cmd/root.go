package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/andrewstucki/consul-services/pkg"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

var (
	tcpServiceCount       int
	httpServiceCount      int
	duplicateServiceCount int
	resourceFolder        string
	consulBinary          string
	runConsul             bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "consul-services",
	Short: "Boots and registers a series of Consul service mesh services used in testing",
	Run: func(cmd *cobra.Command, args []string) {
		retcode := 0
		defer func() { os.Exit(retcode) }()

		logger := hclog.Default()

		config := pkg.RunnerConfig{
			TCPServiceCount:   tcpServiceCount,
			HTTPServiceCount:  httpServiceCount,
			ServiceDuplicates: duplicateServiceCount,
			ResourceFolder:    resourceFolder,
			ConsulBinary:      consulBinary,
			RunConsul:         runConsul,
			Logger:            logger,
		}

		if err := config.Validate(); err != nil {
			logger.Error("Error configuring service runners", "err", err)
			retcode = 1
			return
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		runner := pkg.NewRunner(config)
		if err := runner.Run(ctx); err != nil {
			select {
			case <-ctx.Done():
			default:
				logger.Error("Error running services", "err", err)
				retcode = 1
			}
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().IntVar(&tcpServiceCount, "tcp", 1, "Number of TCP-based services to register on the mesh.")
	rootCmd.Flags().IntVar(&httpServiceCount, "http", 1, "Number of HTTP-based services to register on the mesh.")
	rootCmd.Flags().IntVarP(&duplicateServiceCount, "duplicates", "d", 1, "Number of duplicate services to register on the mesh.")
	rootCmd.Flags().StringVarP(&resourceFolder, "resources", "r", "", "Path to a folder containing extra configuration entries to write.")
	rootCmd.Flags().StringVar(&consulBinary, "consul", "", "Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.")
	rootCmd.Flags().BoolVar(&runConsul, "run", false, "Additionally run Consul binary in agent mode.")
}
