package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/consul/api"
)

// ConsulAgent is a Consul agent in dev mode
type ConsulAgent struct {
	*ConsulCommand

	// Datacenter is the datacenter to register in
	Datacenter string

	// Server used in registering information about the deployed consul instance
	Server *server.Server

	// tracker for allocations
	tracker *tracker
}

// Write writes the Consul agent config
func (c *ConsulAgent) Write() error {
	return c.writeConfig()
}

// Run runs the Consul agent
func (c *ConsulAgent) Run(ctx context.Context) error {
	c.Server.AddConsul(server.Consul{
		Datacenter: c.Datacenter,
		Ports:      c.tracker.ports,
		NamedPorts: c.tracker.namedPorts,
	})

	return c.runConsulBinary(ctx, nil, []string{
		"agent", "-dev",
		"-config-file", c.configFile(),
	})
}

func (c *ConsulAgent) join(ctx context.Context, addresses []string) error {
	filtered := []string{}
	for _, address := range addresses {
		if address == c.wanAddress() {
			continue
		}
		filtered = append(filtered, address)
	}
	return c.runConsulBinary(ctx, nil, append([]string{
		"join",
		"-http-addr", c.address(),
		"-wan",
	}, filtered...))
}

func (c *ConsulAgent) writeConfig() error {
	return c.renderTemplate(agentTemplate, c.configFile())
}

func (c *ConsulAgent) configFile() string {
	return path.Join(c.Folder, fmt.Sprintf("datacenter-%s.hcl", c.Datacenter))
}

func (c *ConsulAgent) ready(ctx context.Context) error {
	client, err := c.client()
	if err != nil {
		return err
	}

	return backoff.Retry(func() error {
		options := &api.QueryOptions{
			Datacenter: c.Datacenter,
		}
		_, meta, err := client.Catalog().Nodes(options.WithContext(ctx))
		if err != nil {
			return err
		}
		if !meta.KnownLeader {
			return errors.New("no known consul leader")
		}
		return nil
	}, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 20), ctx))
}

func (c *ConsulAgent) client() (*api.Client, error) {
	return api.NewClient(&api.Config{
		Address:    c.address(),
		Datacenter: c.Datacenter,
	})
}

func (c *ConsulAgent) address() string {
	return fmt.Sprintf("http://localhost:%d", c.tracker.namedPorts["http"])
}

func (c *ConsulAgent) wanAddress() string {
	return fmt.Sprintf("localhost:%d", c.tracker.namedPorts["serf_wan"])
}

func (c *ConsulAgent) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return os.WriteFile(name, rendered, 0600)
}

type configArgs struct {
	*tracker
	Datacenter string
}

func (c *ConsulAgent) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	if err := getTemplate(name).Execute(&buffer, &configArgs{
		tracker:    c.tracker,
		Datacenter: c.Datacenter,
	}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
