package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/hashicorp/go-hclog"
)

func init() {
	color.NoColor = os.Getenv("NO_COLOR") != ""
}

func createLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Color:                hclog.ForceColor,
		ColorHeaderAndFields: true,
		DisableTime:          true,
		Level:                hclog.Info,
		Output:               os.Stderr,
	})
}
