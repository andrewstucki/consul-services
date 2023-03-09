package pkg

import (
	"errors"
	"fmt"
	"os"

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
	// HTTPServiceCount specifies the number of HTTP-based services to register on the mesh.
	HTTPServiceCount int
	// ExternalTCPServiceCount specifies the number of external TCP-based services to register on the mesh.
	ExternalTCPServiceCount int
	// ExternalHTTPServiceCount specifies the number of external HTTP-based services to register on the mesh.
	ExternalHTTPServiceCount int
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
	// Datacenters specifies the list of datacenters to deploy resources in.
	Datacenters []string
	// Logger specifies the logger to use for output
	Logger hclog.Logger

	// consulCommand interacts with the cached location of the found consul binary
	consulCommand *ConsulCommand
}

// SetLogger resets the underlying logger.
func (c *RunnerConfig) SetLogger(logger hclog.Logger) {
	c.Logger = logger
	c.consulCommand.Logger = logger
}

// Validate validates the runner configuration.
func (c *RunnerConfig) Validate() error {
	if err := c.validateSocket(); err != nil {
		return err
	}

	if err := c.validateConsul(); err != nil {
		return err
	}

	if err := c.validateDatacenters(); err != nil {
		return err
	}

	if err := c.validateResourceFolder(); err != nil {
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

func (c *RunnerConfig) validateDatacenters() error {
	if len(c.Datacenters) == 0 {
		return errors.New("no datacenters specified")
	}

	seen := map[string]struct{}{}
	for _, dc := range c.Datacenters {
		if _, ok := seen[dc]; ok {
			return fmt.Errorf("duplicate datacenter name specified: %q", dc)
		}
		seen[dc] = struct{}{}
	}

	return nil
}

func (c *RunnerConfig) validateResourceFolder() error {
	if c.ResourceFolder == "" {
		return nil
	}

	info, err := os.Stat(c.ResourceFolder)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("--resources must be a directory")
	}

	if len(c.Datacenters) == 1 {
		return nil
	}

	// ensure we have a directory with subfolders aligning with
	// the name of the DC we want them created in

	f, err := os.Open(c.ResourceFolder)
	if err != nil {
		return err
	}
	defer f.Close()

	files, err := f.Readdir(0)
	if err != nil {
		return err
	}

	for _, dc := range c.Datacenters {
		found := false
		for _, file := range files {
			if !file.IsDir() {
				continue
			}

			if file.Name() == dc {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("multiple datacenters specified, but no folder named %q found in resource folder", dc)
		}
	}

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

func (c *RunnerConfig) validateSocket() error {
	_, err := os.Stat(c.Socket)
	if !os.IsNotExist(err) {
		return fmt.Errorf("existing socket found: %q", c.Socket)
	}
	return nil
}
