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
	gatewayFile           string
	extraFilesFolder      string
	runConsul             bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "consul-services",
	Short: "Boots and registers a series of Consul service mesh services used in testing",
	Run: func(cmd *cobra.Command, args []string) {
		logger := hclog.Default()

		config := pkg.RunnerConfig{
			TCPServiceCount:   tcpServiceCount,
			HTTPServiceCount:  httpServiceCount,
			ServiceDuplicates: duplicateServiceCount,
			ResourceFolder:    resourceFolder,
			GatewayFile:       gatewayFile,
			ExtraFilesFolder:  extraFilesFolder,
			ConsulBinary:      consulBinary,
			RunConsul:         runConsul,
			Logger:            logger,
		}

		if err := config.Validate(); err != nil {
			logger.Error("Error configuring service runners", "err", err)
			os.Exit(1)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		runner := pkg.NewRunner(config)
		if err := runner.Run(ctx); err != nil {
			select {
			case <-ctx.Done():
			default:
				logger.Error("Error running services", "err", err)
				os.Exit(1)
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
	rootCmd.Flags().StringVarP(&resourceFolder, "resources", "r", "", "Folder of resources to apply, overrides tcp and http flags.")
	rootCmd.Flags().StringVar(&consulBinary, "consul", "", "Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.")
	rootCmd.Flags().StringVar(&gatewayFile, "gateway", "", "Path to gateway definition to create, filed should be named 'api.hcl', 'ingress.hcl', etc. with a Port interpolation.")
	rootCmd.Flags().StringVarP(&extraFilesFolder, "extras", "e", "", "Path to a folder containing extra configuration entries to write.")
	rootCmd.Flags().BoolVar(&runConsul, "run", false, "Additionally run Consul binary in agent mode.")
}
