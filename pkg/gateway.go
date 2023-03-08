package pkg

import (
	"context"
	"fmt"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/hashicorp/consul/api"
)

var knownGateways = map[string]struct{}{
	api.APIGateway:                     {},
	api.IngressGateway:                 {},
	api.TerminatingGateway:             {},
	string(api.ServiceKindMeshGateway): {},
}

// ConsulGateway is a gateway service to run on the Consul service mesh using the `consul connect envoy` command.
type ConsulGateway struct {
	*ConsulConfigEntry

	// Name is the name of the gateway to run
	Name string
	// Kind is the kind of gateway to run
	Kind string
	// DefinitionFile is the path to the file used for registering the gateway
	DefinitionFile string
	// Server is used for service registration
	Server *server.Server

	// adminPort is the port allocated for envoy's admin interface
	adminPort int
}

// Run runs the Consul gateway
func (c *ConsulGateway) Run(ctx context.Context) error {
	if err := c.allocatePorts(); err != nil {
		return err
	}

	if err := c.Write(ctx); err != nil {
		return err
	}

	return c.runEnvoy(ctx)
}

func (c *ConsulGateway) allocatePorts() error {
	adminPort, err := freePort()
	if err != nil {
		return err
	}

	c.adminPort = adminPort
	return nil
}

func (c *ConsulGateway) runEnvoy(ctx context.Context) error {
	c.Logger.Info("running gateway", "admin", c.adminPort, "ports", c.tracker.ports)

	c.Server.Register(server.Service{
		Kind:  c.Kind,
		Name:  c.Name,
		Ports: c.tracker.ports,
	})

	return c.runConsulBinary(ctx, c.gatewayArgs())
}

func (c *ConsulGateway) gatewayArgs() []string {
	args := []string{
		"connect", "envoy",
		"-gateway", c.Kind,
		"-register",
		"-service", c.Name,
		"-proxy-id", c.Name,
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", c.adminPort),
	}

	if len(c.tracker.ports) > 0 && c.Kind == "api" {
		args = append(args, "-address", fmt.Sprintf("127.0.0.1:%d", c.tracker.ports[0]))
	}
	return args
}
