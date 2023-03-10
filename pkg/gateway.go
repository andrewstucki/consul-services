package pkg

import (
	"context"

	"github.com/andrewstucki/consul-services/pkg/commands"
	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/hashicorp/consul/api"
)

var knownGateways = map[string]string{
	api.APIGateway:                     "api",
	api.IngressGateway:                 "ingress",
	api.TerminatingGateway:             "terminating",
	string(api.ServiceKindMeshGateway): "mesh",
}

// ConsulGateway is a gateway service to run on the Consul service mesh using the `consul connect envoy` command.
type ConsulGateway struct {
	*ConsulConfigEntry

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

func (c *ConsulGateway) gatewayKind() string {
	return knownGateways[c.Kind]
}

func (c *ConsulGateway) allocatePorts() error {
	adminPort, err := freePort()
	if err != nil {
		return err
	}

	if c.gatewayKind() != "api" {
		// we allocate an additional port here and grab
		// it as the first allocation in our registration so that
		// we can run all the gateways on different ports
		// rather than just defaulting to 8443
		_, err := c.tracker.GetPort()
		if err != nil {
			return err
		}
	}

	c.adminPort = adminPort
	return nil
}

func (c *ConsulGateway) runEnvoy(ctx context.Context) error {
	c.Logger.Info("running gateway", "admin", c.adminPort, "ports", c.tracker.ports)

	registrationPort := 8443
	if len(c.tracker.ports) > 0 {
		registrationPort = c.tracker.ports[0]
	}

	return c.runConsulBinary(ctx, func(log string) {
		c.Server.Register(server.Service{
			Datacenter:     c.locality.Datacenter,
			Partition:      c.locality.Partition,
			Namespace:      c.locality.Namespace,
			Kind:           c.Kind,
			Name:           c.Name,
			AdminPort:      c.adminPort,
			Ports:          c.tracker.ports,
			NamedPorts:     c.tracker.namedPorts,
			Logs:           log,
			ConsulAddress:  c.locality.getAddress(),
			RegisteredPort: registrationPort,
		})
	}, commands.GatewayRegistrationArgs(c.gatewayKind(), c.Name, c.locality.getAddress(), c.adminPort, registrationPort))
}
