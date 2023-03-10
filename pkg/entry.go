package pkg

import (
	"bytes"
	"context"
	"os"
	"path"
	"text/template"

	"github.com/andrewstucki/consul-services/pkg/commands"
	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/andrewstucki/consul-services/pkg/vfs"
)

// ConsulConfigEntry is a config entry to write to Consul
type ConsulConfigEntry struct {
	*ConsulCommand

	// Kind string is the kind in the file definition
	Kind string

	// Name is the name in the file definition
	Name string

	// DefinitionFile is the path to the file used for writing the config entry
	DefinitionFile string

	// Server is the server to register the config entry with
	Server *server.Server

	// tracker holds the allocated information for the entry
	tracker *tracker
	// locality identifies the datacenter/partition/namespace an entry is written to
	locality locality
}

func (c *ConsulConfigEntry) renderedFile() string {
	return path.Join(c.locality.Datacenter, "entries", path.Base(c.DefinitionFile))
}

func (c *ConsulConfigEntry) Write(ctx context.Context) error {
	c.Logger.Info("writing config entry definition", "file", c.DefinitionFile)

	if err := c.renderTemplate(c.DefinitionFile, c.renderedFile()); err != nil {
		return err
	}

	return c.runConsulBinary(ctx, func(log string) {
		c.Server.AddEntry(server.Entry{
			Datacenter:    c.locality.Datacenter,
			Partition:     c.locality.Partition,
			Namespace:     c.locality.Namespace,
			Kind:          c.Kind,
			Name:          c.Name,
			File:          c.renderedFile(),
			ConsulAddress: c.locality.getAddress(),
		})
	}, commands.WriteConfigArgs(
		c.locality.Datacenter,
		c.locality.getAddress(),
		vfs.PathFor(c.renderedFile()),
	))
}

func (c *ConsulConfigEntry) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return vfs.WriteFile(name, rendered, 0600)
}

func (c *ConsulConfigEntry) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	file, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	template, err := template.New(name).Parse(string(file))
	if err != nil {
		return nil, err
	}

	if err := template.Execute(&buffer, c.tracker); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
