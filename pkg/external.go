package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/andrewstucki/consul-services/pkg/commands"
	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/andrewstucki/consul-services/pkg/vfs"
	"github.com/hashicorp/consul/api"
)

// ConsulExternalService is a service run as an external service in the Consul service mesh.
type ConsulExternalService struct {
	*ConsulCommand

	// ID is the id of the service to run
	ID string
	// Name is the name of the service to run
	Name string
	// Protocol is the protocol of the service
	Protocol string
	// OnRegister is a channel to write back to when we've registered our services
	OnRegister chan struct{}
	// Server is used for service registration
	Server *server.Server

	// servicePort is the port allocated for the service
	servicePort int
	// tracker holds any dynamic allocations
	tracker *tracker

	// locality identifies the datacenter/partition/namespace a service is deployed in
	locality locality
}

// Run runs the Consul external service
func (c *ConsulExternalService) Run(ctx context.Context) error {
	var err error

	c.servicePort, err = freePort()
	if err != nil {
		return err
	}

	if err := c.renderServiceDefaults(); err != nil {
		return err
	}
	if err := c.renderService(); err != nil {
		return err
	}

	if err := c.registerService(ctx); err != nil {
		return err
	}
	if err := c.writeServiceDefaults(ctx); err != nil {
		return err
	}

	c.OnRegister <- struct{}{}
	c.Server.Register(server.Service{
		Datacenter:              c.locality.Datacenter,
		Partition:               c.locality.Partition,
		Namespace:               c.locality.Namespace,
		Kind:                    "external",
		Name:                    c.ID,
		Ports:                   []int{c.servicePort},
		ServiceDefaultsFile:     c.serviceDefaultsFile(),
		ServiceRegistrationFile: c.serviceFile(),
		Protocol:                c.Protocol,
		ConsulAddress:           c.locality.getAddress(),
	})

	return c.runService(ctx)
}

func (c *ConsulExternalService) renderService() error {
	return c.renderTemplate(externalServiceTemplate, c.serviceFile())
}

func (c *ConsulExternalService) registerService(ctx context.Context) error {
	c.Logger.Info("registering service", "id", c.ID)

	client, err := c.locality.getClient()
	if err != nil {
		return err
	}

	data, err := vfs.ReadFile(c.serviceFile())
	if err != nil {
		return err
	}

	registration := &api.CatalogRegistration{}
	if err := json.Unmarshal(data, registration); err != nil {
		return err
	}

	options := &api.WriteOptions{
		Datacenter: c.locality.Datacenter,
	}
	if _, err := client.Catalog().Register(registration, options.WithContext(ctx)); err != nil {
		return err
	}

	return nil
}

func (c *ConsulExternalService) writeServiceDefaults(ctx context.Context) error {
	c.Logger.Info("writing service defaults", "id", c.ID)

	return c.runConsulBinary(ctx, nil, commands.WriteConfigArgs(
		c.locality.Datacenter,
		c.locality.getAddress(),
		vfs.PathFor(c.serviceDefaultsFile()),
	))
}

func (c *ConsulExternalService) serviceFile() string {
	return path.Join(c.locality.Datacenter, fmt.Sprintf("external-service-%s.json", c.ID))
}

func (c *ConsulExternalService) serviceDefaultsFile() string {
	return path.Join(c.locality.Datacenter, fmt.Sprintf("external-service-defaults-%s.hcl", c.ID))
}

func (c *ConsulExternalService) renderServiceDefaults() error {
	return c.renderTemplate(serviceDefaultsTemplate, c.serviceDefaultsFile())
}

func (c *ConsulExternalService) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return vfs.WriteFile(name, rendered, 0600)
}

func (c *ConsulExternalService) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	if err := getTemplate(name).Execute(&buffer, &templateArgs{
		tracker:     c.tracker,
		ID:          c.ID,
		Name:        c.Name,
		Protocol:    c.Protocol,
		ServicePort: c.servicePort,
	}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (c *ConsulExternalService) runService(ctx context.Context) error {
	c.Logger.Info("running service", "protocol", c.Protocol, "service", c.servicePort)

	service := &Service{
		ID:       c.ID,
		Protocol: c.Protocol,
		Port:     c.servicePort,
	}

	return service.Run(ctx)
}
