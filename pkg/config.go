package pkg

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

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
	// ResourceFolder specifies a folder of resources to apply, overriding any specified number of TCP or HTTP services.
	ResourceFolder string
	// ExtraFilesFolder specifies a folder of additional config entries to apply.
	ExtraFilesFolder string
	// GatewayFile specifies the file to a gateway definition that you want to deploy.
	GatewayFile string
	// ConsulBinary specifies the Consul binary to use for running services.
	ConsulBinary string
	// RunConsul specifies whether a Consul agent in dev mode should also be run
	RunConsul bool
	// Logger specifies the logger to use for output
	Logger hclog.Logger

	// consulBinary is the cached location of the found consul binary
	consulBinary string
}

// Validate validates the runner configuration.
func (c *RunnerConfig) Validate() error {
	if err := c.validateConsul(); err != nil {
		return err
	}

	if c.ResourceFolder != "" {
		return c.validateResourceFolder()
	}

	return c.validateServiceCounts()
}

func (c *RunnerConfig) validateConsul() error {
	paths := []string{c.ConsulBinary, defaultBinaryPath}
	path, err := exec.LookPath(binaryName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		paths = append(paths, path)
	}

	return c.findConsul(paths...)
}

func (c *RunnerConfig) findConsul(paths ...string) error {
	for _, path := range paths {
		if path == "" {
			continue
		}

		found, normalized, err := checkConsulExecutable(path)
		if err != nil {
			return err
		}
		if found {
			c.consulBinary = normalized
			return nil
		}
	}
	return errors.New("consul binary not found")
}

func (c *RunnerConfig) validateResourceFolder() error {
	info, err := os.Stat(c.ResourceFolder)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("folder %q is not a directory", c.ResourceFolder)
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

func checkConsulExecutable(path string) (bool, string, error) {
	info, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return false, "", err
	}

	if info == nil {
		return false, "", nil
	}

	normalized, err := filepath.Abs(path)
	if err != nil {
		return false, "", err
	}

	return err == nil && isExecutable(info), normalized, nil
}

func isExecutable(info fs.FileInfo) bool {
	return !info.IsDir() && (info.Mode()&0111 != 0)
}
