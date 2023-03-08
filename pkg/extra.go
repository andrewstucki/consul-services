package pkg

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	"github.com/hashicorp/go-hclog"
)

// ConsulConfigEntry is a config entry to write to Consul
type ConsulConfigEntry struct {
	// ConsulBinary is the path on the system to the Consul binary used to invoke registration and connect commands.
	ConsulBinary string
	// DefinitionFile is the path to the file used for writing the config entry
	DefinitionFile string
	// Logger is the logger used for logging messages
	Logger hclog.Logger
}

func (c *ConsulConfigEntry) Write(ctx context.Context) error {
	c.Logger.Info("writing config entry definition", "file", c.DefinitionFile)

	return c.runConsulBinary(ctx, c.configEntryArgs())
}

func (c *ConsulConfigEntry) runConsulBinary(ctx context.Context, args []string) error {
	var errBuffer bytes.Buffer

	cmd := exec.CommandContext(ctx, c.ConsulBinary, args...)
	cmd.Stderr = &errBuffer

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New(errBuffer.String())
		}
		return err
	}

	return nil
}

func (c *ConsulConfigEntry) configEntryArgs() []string {
	return []string{
		"config", "write",
		c.DefinitionFile,
	}
}
