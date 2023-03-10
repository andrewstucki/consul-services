package pkg

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/andrewstucki/consul-services/pkg/commands"
	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/andrewstucki/consul-services/pkg/vfs"
	"golang.org/x/sync/errgroup"
)

// ConsulMeshService is a service run on the Consul service mesh using the `consul connect envoy` command.
type ConsulMeshService struct {
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
	// ExternalUpstreams are the external services to add upstreams for
	ExternalUpstreams []string

	// adminPort is the port allocated for envoy's admin interface
	adminPort int
	// proxyPort is the port allocated for envoy's proxy interface
	proxyPort int
	// servicePort is the port allocated for the service
	servicePort int
	// tracker holds any dynamic allocations
	tracker *tracker

	// locality identifies the datacenter/partition/namespace a service is deployed in
	locality locality
}

// Run runs the Consul mesh service
func (c *ConsulMeshService) Run(ctx context.Context) error {
	if err := c.allocatePorts(); err != nil {
		return err
	}

	if err := c.renderServiceDefaults(); err != nil {
		return err
	}
	if err := c.renderService(); err != nil {
		return err
	}
	if err := c.renderServiceProxy(); err != nil {
		return err
	}

	if err := c.registerService(ctx); err != nil {
		return err
	}
	if err := c.registerServiceProxy(ctx); err != nil {
		return err
	}
	if err := c.writeServiceDefaults(ctx); err != nil {
		return err
	}

	c.OnRegister <- struct{}{}
	c.Server.Register(server.Service{
		Datacenter: c.locality.Datacenter,
		Partition:  c.locality.Partition,
		Namespace:  c.locality.Namespace,
		Kind:       "service",
		Name:       c.ID,
		Ports:      []int{c.servicePort},
	})

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return c.runEnvoy(ctx)
	})
	group.Go(func() error {
		return c.runService(ctx)
	})

	return group.Wait()
}

func (c *ConsulMeshService) allocatePorts() error {
	adminPort, err := freePort()
	if err != nil {
		return err
	}
	proxyPort, err := freePort()
	if err != nil {
		return err
	}
	servicePort, err := freePort()
	if err != nil {
		return err
	}

	c.adminPort = adminPort
	c.proxyPort = proxyPort
	c.servicePort = servicePort
	return nil
}

func (c *ConsulMeshService) registerService(ctx context.Context) error {
	c.Logger.Info("registering service", "id", c.ID)

	return c.runConsulBinary(ctx, nil, commands.RegisterServiceArgs(
		c.locality.Datacenter,
		c.locality.getAddress(),
		vfs.PathFor(c.serviceFile()),
	))
}

func (c *ConsulMeshService) registerServiceProxy(ctx context.Context) error {
	c.Logger.Info("registering sidecar proxy", "id", c.ID)

	return c.runConsulBinary(ctx, nil, commands.RegisterServiceArgs(
		c.locality.Datacenter,
		c.locality.getAddress(),
		vfs.PathFor(c.serviceProxyFile()),
	))
}

func (c *ConsulMeshService) writeServiceDefaults(ctx context.Context) error {
	c.Logger.Info("writing service defaults", "id", c.ID)

	return c.runConsulBinary(ctx, nil, commands.WriteConfigArgs(
		c.locality.Datacenter,
		c.locality.getAddress(),
		vfs.PathFor(c.serviceDefaultsFile()),
	))
}

func (c *ConsulMeshService) runService(ctx context.Context) error {
	c.Logger.Info("running service", "protocol", c.Protocol, "admin", c.adminPort, "service", c.servicePort, "proxy", c.proxyPort)

	service := &Service{
		ID:       c.ID,
		Protocol: c.Protocol,
		Port:     c.servicePort,
	}

	return service.Run(ctx)
}

func (c *ConsulMeshService) runEnvoy(ctx context.Context) error {
	c.Logger.Info("running sidecar")

	return c.runConsulBinary(ctx, func(log string) {
		c.Server.Register(server.Service{
			Datacenter:              c.locality.Datacenter,
			Partition:               c.locality.Partition,
			Namespace:               c.locality.Namespace,
			Kind:                    "connect-proxy",
			Name:                    c.ID + "-proxy",
			AdminPort:               c.adminPort,
			Ports:                   append([]int{c.proxyPort}, c.tracker.ports...),
			NamedPorts:              c.tracker.namedPorts,
			Logs:                    log,
			ServiceDefaultsFile:     c.serviceDefaultsFile(),
			ServiceProxyFile:        c.serviceProxyFile(),
			ServiceRegistrationFile: c.serviceFile(),
			ConsulAddress:           c.locality.getAddress(),
			Protocol:                c.Protocol,
			ServicePort:             c.servicePort,
		})
	}, commands.SidecarArgs(
		c.locality.getAddress(),
		c.ID,
		c.adminPort,
	))
}

func (c *ConsulMeshService) renderService() error {
	return c.renderTemplate(serviceTemplate, c.serviceFile())
}

func (c *ConsulMeshService) renderServiceDefaults() error {
	return c.renderTemplate(serviceDefaultsTemplate, c.serviceDefaultsFile())
}

func (c *ConsulMeshService) renderServiceProxy() error {
	return c.renderTemplate(serviceProxyTemplate, c.serviceProxyFile())
}

func (c *ConsulMeshService) renderTemplate(template, name string) error {
	rendered, err := c.executeTemplate(template)
	if err != nil {
		return err
	}
	return vfs.WriteFile(name, rendered, 0600)
}

func (c *ConsulMeshService) serviceFile() string {
	return path.Join(c.locality.Datacenter, fmt.Sprintf("service-%s.hcl", c.ID))
}

func (c *ConsulMeshService) serviceDefaultsFile() string {
	return path.Join(c.locality.Datacenter, fmt.Sprintf("service-defaults-%s.hcl", c.ID))
}

func (c *ConsulMeshService) serviceProxyFile() string {
	return path.Join(c.locality.Datacenter, fmt.Sprintf("service-proxy-%s.hcl", c.ID))
}

func (c *ConsulMeshService) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	if err := getTemplate(name).Execute(&buffer, &templateArgs{
		tracker:           c.tracker,
		ID:                c.ID,
		Name:              c.Name,
		Protocol:          c.Protocol,
		ServicePort:       c.servicePort,
		ProxyPort:         c.proxyPort,
		ExternalUpstreams: c.ExternalUpstreams,
	}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
