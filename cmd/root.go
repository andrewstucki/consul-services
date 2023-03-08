package cmd

import (
	"context"
	"os"
	"os/signal"
	"path"

	"github.com/andrewstucki/consul-services/pkg"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigFilename = ".consul-services.yaml"
)

var (
	defaultUnixSocket string

	tcpServiceCount       int
	httpServiceCount      int
	duplicateServiceCount int
	resourceFolder        string
	consulBinary          string
	socket                string
	configFile            string
	runConsul             bool
)

func setCommandFlag(cmd *cobra.Command, flag string) {
	if value := viper.GetString(flag); value != "" {
		cmd.Flags().Set(flag, value)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "consul-services",
	Short: "Boots and registers a series of Consul service mesh services used in testing",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if socket == "" {
			socket = defaultUnixSocket
		}
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if configFile != "" {
			viper.SetConfigFile(configFile)
		}

		err := viper.ReadInConfig()
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		}

		setCommandFlag(cmd, "tcp")
		setCommandFlag(cmd, "http")
		setCommandFlag(cmd, "duplicates")
		setCommandFlag(cmd, "resources")
		setCommandFlag(cmd, "consul")
		setCommandFlag(cmd, "socket")
		setCommandFlag(cmd, "run")

		return nil
	},
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
			Socket:            socket,
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
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	defaultUnixSocket = path.Join(home, ".consul-services.sock")

	rootCmd.Flags().IntVar(&tcpServiceCount, "tcp", 1, "Number of TCP-based services to register on the mesh.")
	viper.BindPFlag("tcp", rootCmd.Flags().Lookup("tcp"))
	rootCmd.Flags().IntVar(&httpServiceCount, "http", 1, "Number of HTTP-based services to register on the mesh.")
	viper.BindPFlag("http", rootCmd.Flags().Lookup("http"))
	rootCmd.Flags().IntVarP(&duplicateServiceCount, "duplicates", "d", 1, "Number of duplicate services to register on the mesh.")
	viper.BindPFlag("duplicates", rootCmd.Flags().Lookup("duplicates"))
	rootCmd.Flags().StringVarP(&resourceFolder, "resources", "r", "", "Path to a folder containing extra configuration entries to write.")
	viper.BindPFlag("resources", rootCmd.Flags().Lookup("resources"))
	rootCmd.Flags().StringVar(&consulBinary, "consul", "", "Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.")
	viper.BindPFlag("consul", rootCmd.Flags().Lookup("consul"))
	rootCmd.PersistentFlags().StringVarP(&socket, "socket", "s", "", "Path to unix socket for control server. (default \"$HOME/.consul-services.sock\")")
	viper.BindPFlag("socket", rootCmd.PersistentFlags().Lookup("socket"))
	rootCmd.Flags().BoolVar(&runConsul, "run", false, "Additionally run Consul binary in agent mode.")
	viper.BindPFlag("run", rootCmd.Flags().Lookup("run"))

	rootCmd.Flags().StringVarP(&configFile, "config", "c", defaultConfigFilename, "Path to configuration file.")

	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
}
