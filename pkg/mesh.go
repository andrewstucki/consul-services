package pkg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"

	"github.com/andrewstucki/consul-services/pkg/server"
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

	// adminPort is the port allocated for envoy's admin interface
	adminPort int
	// proxyPort is the port allocated for envoy's proxy interface
	proxyPort int
	// servicePort is the port allocated for the service
	servicePort int
	// tracker holds any dynamic allocations
	tracker *tracker
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
		Kind:  "service",
		Name:  c.ID,
		Ports: []int{c.servicePort},
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
	c.Logger.Info("registering service")

	return c.runConsulBinary(ctx, nil, c.serviceArgs())
}

func (c *ConsulMeshService) registerServiceProxy(ctx context.Context) error {
	c.Logger.Info("registering sidecar proxy")

	return c.runConsulBinary(ctx, nil, c.serviceProxyArgs())
}

func (c *ConsulMeshService) writeServiceDefaults(ctx context.Context) error {
	c.Logger.Info("writing service defaults")

	return c.runConsulBinary(ctx, nil, c.serviceDefaultsArgs())
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
			Kind:       "connect-proxy",
			Name:       c.ID + "-proxy",
			AdminPort:  c.adminPort,
			Ports:      append([]int{c.proxyPort}, c.tracker.ports...),
			NamedPorts: c.tracker.namedPorts,
			Logs:       log,
		})
	}, c.sidecarArgs())
}

func (c *ConsulMeshService) serviceArgs() []string {
	return []string{
		"services", "register",
		c.serviceFile(),
	}
}

func (c *ConsulMeshService) serviceProxyArgs() []string {
	return []string{
		"services", "register",
		c.serviceProxyFile(),
	}
}

func (c *ConsulMeshService) serviceDefaultsArgs() []string {
	return []string{
		"config", "write",
		c.serviceDefaultsFile(),
	}
}

func (c *ConsulMeshService) sidecarArgs() []string {
	return []string{
		"connect", "envoy",
		"-sidecar-for", c.ID,
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", c.adminPort),
		"--", "-l", "trace",
	}
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
	return os.WriteFile(name, rendered, 0600)
}

func (c *ConsulMeshService) serviceFile() string {
	return path.Join(c.Folder, fmt.Sprintf("service-%s.hcl", c.ID))
}

func (c *ConsulMeshService) serviceDefaultsFile() string {
	return path.Join(c.Folder, fmt.Sprintf("service-defaults-%s.hcl", c.ID))
}

func (c *ConsulMeshService) serviceProxyFile() string {
	return path.Join(c.Folder, fmt.Sprintf("service-proxy-%s.hcl", c.ID))
}

func (c *ConsulMeshService) executeTemplate(name string) ([]byte, error) {
	var buffer bytes.Buffer

	if err := getTemplate(name).Execute(&buffer, &templateArgs{
		tracker:     c.tracker,
		ID:          c.ID,
		Name:        c.Name,
		Protocol:    c.Protocol,
		ServicePort: c.servicePort,
		ProxyPort:   c.proxyPort,
	}); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
