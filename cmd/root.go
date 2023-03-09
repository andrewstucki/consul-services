package cmd

import (
	"context"
	"os"
	"os/signal"
	"path"
	"strconv"

	"github.com/andrewstucki/consul-services/pkg"
	"github.com/andrewstucki/consul-services/pkg/daemonize"
	"github.com/docker/docker/pkg/reexec"
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultConfigFilename = ".consul-services.yaml"
)

var (
	defaultUnixSocket string

	tcpServiceCount          int
	httpServiceCount         int
	httpExternalServiceCount int
	tcpExternalServiceCount  int
	duplicateServiceCount    int
	resourceFolder           string
	consulBinary             string
	socket                   string
	output                   string
	configFile               string
	datacenters              []string
	runConsul                bool
	daemonizeRunner          bool
)

func setCommandFlag(cmd *cobra.Command, flag string) {
	if value := viper.GetString(flag); value != "" {
		cmd.Flags().Set(flag, value)
	}
}

func setCommandFlagExtended(cmd *cobra.Command, viperFlag, cobraFlag string) {
	if value := viper.GetString(viperFlag); value != "" {
		cmd.Flags().Set(cobraFlag, value)
	}
}

func setCommandFlagArray(cmd *cobra.Command, viperFlag, cobraFlag string) {
	for _, viperValue := range viper.GetStringSlice(viperFlag) {
		cmd.Flags().Set(cobraFlag, viperValue)
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

		setCommandFlag(cmd, "duplicates")
		setCommandFlag(cmd, "resources")
		setCommandFlag(cmd, "consul")
		setCommandFlag(cmd, "socket")
		setCommandFlag(cmd, "run")

		setCommandFlagArray(cmd, "datacenters", "datacenter")
		setCommandFlagExtended(cmd, "services.tcp", "tcp")
		setCommandFlagExtended(cmd, "services.http", "http")
		setCommandFlagExtended(cmd, "services.external.tcp", "external-tcp")
		setCommandFlagExtended(cmd, "services.external.http", "external-http")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		retcode := 0
		defer func() { os.Exit(retcode) }()

		logger := createLogger()

		config := pkg.RunnerConfig{
			TCPServiceCount:          tcpServiceCount,
			HTTPServiceCount:         httpServiceCount,
			ExternalTCPServiceCount:  tcpExternalServiceCount,
			ExternalHTTPServiceCount: httpExternalServiceCount,
			ServiceDuplicates:        duplicateServiceCount,
			ResourceFolder:           resourceFolder,
			ConsulBinary:             consulBinary,
			Socket:                   socket,
			RunConsul:                runConsul,
			Datacenters:              datacenters,
			Logger:                   logger,
		}

		if err := config.Validate(); err != nil {
			logger.Error("error configuring service runners", "err", err)
			retcode = 1
			return
		}

		// run the daemonization after validation
		// so we know we're likely to succeed at running
		// the child processes
		if daemonizeRunner {
			if err := daemonize.Daemonize(daemonArgs()...); err != nil {
				logger.Error("could not daemonize process", "err", err)
				retcode = 1
			}
			return
		}

		// set the actual output here
		if output != "" {
			sink, err := os.Create(output)
			if err != nil {
				logger.Error("error opening up output sink", "err", err)
				os.Exit(1)
			}
			defer sink.Close()

			logger.Info("redirecting output", "file", output)

			config.SetLogger(hclog.New(&hclog.LoggerOptions{
				Output: sink,
			}))
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		runner := pkg.NewRunner(config)
		if err := runner.Run(ctx); err != nil {
			select {
			case <-ctx.Done():
			default:
				config.Logger.Error("Error running services", "err", err)
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
	if err == nil {
		defaultUnixSocket = path.Join(home, ".consul-services.sock")
	}

	rootCmd.Flags().IntVar(&tcpServiceCount, "tcp", 0, "Number of TCP-based services to register on the mesh.")
	viper.BindPFlag("services.tcp", rootCmd.Flags().Lookup("tcp"))
	rootCmd.Flags().IntVar(&httpServiceCount, "http", 1, "Number of HTTP-based services to register on the mesh.")
	viper.BindPFlag("services.http", rootCmd.Flags().Lookup("http"))
	rootCmd.Flags().IntVar(&httpExternalServiceCount, "external-http", 0, "Number of HTTP-based external services to register on the mesh.")
	viper.BindPFlag("services.external.http", rootCmd.Flags().Lookup("external-http"))
	rootCmd.Flags().IntVar(&tcpExternalServiceCount, "external-tcp", 0, "Number of TCP-based external services to register on the mesh.")
	viper.BindPFlag("services.external.tcp", rootCmd.Flags().Lookup("external-tcp"))
	rootCmd.Flags().IntVarP(&duplicateServiceCount, "duplicates", "D", 1, "Number of duplicate services to register on the mesh.")
	viper.BindPFlag("duplicates", rootCmd.Flags().Lookup("duplicates"))
	rootCmd.Flags().StringVarP(&resourceFolder, "resources", "r", "", "Path to a folder containing extra configuration entries to write.")
	viper.BindPFlag("resources", rootCmd.Flags().Lookup("resources"))
	rootCmd.Flags().StringVar(&consulBinary, "consul", "", "Consul binary to use for registration, defaults to a binary found in the current folder and then the PATH.")
	viper.BindPFlag("consul", rootCmd.Flags().Lookup("consul"))
	rootCmd.PersistentFlags().StringVarP(&socket, "socket", "s", "", "Path to unix socket for control server. (default \"$HOME/.consul-services.sock\")")
	viper.BindPFlag("socket", rootCmd.PersistentFlags().Lookup("socket"))
	rootCmd.Flags().BoolVar(&runConsul, "run", false, "Additionally run Consul binary in agent mode.")
	viper.BindPFlag("run", rootCmd.Flags().Lookup("run"))
	rootCmd.Flags().StringArrayVar(&datacenters, "datacenter", []string{"dc1"}, "Datacenters to deploy into.")
	viper.BindPFlag("datacenters", rootCmd.Flags().Lookup("datacenter"))
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Path to use for output rather than stdout.")
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	rootCmd.Flags().StringVarP(&configFile, "config", "c", defaultConfigFilename, "Path to configuration file.")
	rootCmd.Flags().BoolVarP(&daemonizeRunner, "daemon", "d", false, "Daemonize the process.")
	daemonize.SetDaemonizationFlag("--daemon", "-d")

	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")
}

func daemonArgs() []string {
	daemonOut := output
	if daemonOut == "" {
		daemonOut = "daemon.log"
	}

	args := []string{
		reexec.Self(),
		"--tcp", strconv.Itoa(tcpServiceCount),
		"--http", strconv.Itoa(httpServiceCount),
		"--duplicates", strconv.Itoa(duplicateServiceCount),
		"--resources", resourceFolder,
		"--socket", socket,
		"--config", configFile,
		"--consul", consulBinary,
		"--output", daemonOut,
	}
	if runConsul {
		args = append(args, "--run")
	}

	return args
}
