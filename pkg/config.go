package pkg

import (
	"errors"

	"github.com/hashicorp/go-hclog"
)

const (
	defaultBinaryPath = "consul"
	binaryName        = "consul"
)

// RunnerConfig configures a service runner
type RunnerConfig struct {
	// TCPServiceCount specifies the number of TCP-based services to register on the mesh.
	TCPServiceCount int
	// HTTPServiceCount specifies the number of TCP-based services to register on the mesh.
	HTTPServiceCount int
	// ServiceDuplicates is the amount of times a service should be duplicated (i.e. have the same
	// service name, but different ids)
	ServiceDuplicates int
	// ResourceFolder specifies a folder of additional config entries to apply.
	ResourceFolder string
	// ConsulBinary specifies the Consul binary to use for running services.
	ConsulBinary string
	// Socket specifies the unix socket that the control server serves traffic on.
	Socket string
	// RunConsul specifies whether a Consul agent in dev mode should also be run
	RunConsul bool
	// Logger specifies the logger to use for output
	Logger hclog.Logger

	// consulCommand interacts with the cached location of the found consul binary
	consulCommand *ConsulCommand
}

// Validate validates the runner configuration.
func (c *RunnerConfig) Validate() error {
	if err := c.validateConsul(); err != nil {
		return err
	}

	return c.validateServiceCounts()
}

func (c *RunnerConfig) validateConsul() error {
	consul, err := newCommand(c.ConsulBinary, c.Logger)
	if err != nil {
		return err
	}
	c.consulCommand = consul
	return nil
}

func (c *RunnerConfig) validateServiceCounts() error {
	if c.TCPServiceCount <= 0 && c.HTTPServiceCount <= 0 {
		return errors.New("service counts must be greater than or equal to 1")
	}
	if c.ServiceDuplicates <= 0 {
		return errors.New("service duplicates must be greater than or equal to 1")
	}
	return nil
}
