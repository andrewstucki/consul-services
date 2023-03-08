package main

import (
	"os"

	"github.com/andrewstucki/consul-services/cmd"
	"github.com/andrewstucki/consul-services/pkg/daemonize"
	"github.com/hashicorp/go-hclog"
)

func main() {
	logger := hclog.Default()
	handled, err := daemonize.Handle(os.Args...)
	if err != nil {
		logger.Error("error setting up daemon", "err", err)
		os.Exit(1)
	}
	if !handled {
		cmd.Execute()
	}
}
