package pkg

import (
	"bytes"
	"context"
	"os"
	"path"
	"text/template"
)

// ConsulConfigEntry is a config entry to write to Consul
type ConsulConfigEntry struct {
	*ConsulCommand

	// DefinitionFile is the path to the file used for writing the config entry
	DefinitionFile string

	// tracker holds the allocated information for the entry
	tracker *tracker
}

func (c *ConsulConfigEntry) renderedFile() string {
	return path.Join(c.Folder, path.Base(c.DefinitionFile))
}

func (c *ConsulConfigEntry) Write(ctx context.Context) error {
	c.Logger.Info("writing config entry definition", "file", c.DefinitionFile)

	if err := c.renderTemplate(c.DefinitionFile, c.renderedFile()); err != nil {
		return err
	}

	return c.runConsulBinary(ctx, nil, []string{
		"config", "write",
		c.renderedFile(),
	})
}

func (c *ConsulConfigEntry) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return os.WriteFile(name, rendered, 0600)
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
