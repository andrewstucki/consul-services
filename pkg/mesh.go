package pkg

import (
	"context"
	"fmt"

	"github.com/andrewstucki/consul-services/pkg/server"
)

// ConsulMeshGateway is a gateway service to run on the Consul service mesh using the `consul connect envoy` command.
type ConsulMeshGateway struct {
	*ConsulCommand

	// Server is used for service registration
	Server *server.Server

	// adminPort is the port allocated for envoy's admin interface
	adminPort int
	// proxyPort is the port allocated for envoy's proxy interface
	proxyPort int

	// locality identifies the datacenter/partition/namespace a service is deployed in
	locality locality
}

// Run runs the Consul gateway
func (c *ConsulMeshGateway) Run(ctx context.Context) error {
	if err := c.allocatePorts(); err != nil {
		return err
	}

	return c.runEnvoy(ctx)
}

func (c *ConsulMeshGateway) allocatePorts() error {
	adminPort, err := freePort()
	if err != nil {
		return err
	}

	proxyPort, err := freePort()
	if err != nil {
		return err
	}

	c.adminPort = adminPort
	c.proxyPort = proxyPort
	return nil
}

func (c *ConsulMeshGateway) runEnvoy(ctx context.Context) error {
	c.Logger.Info("running mesh gateway", "admin", c.adminPort)

	return c.runConsulBinary(ctx, func(log string) {
		c.Server.Register(server.Service{
			Datacenter: c.locality.Datacenter,
			Partition:  c.locality.Partition,
			Namespace:  c.locality.Namespace,
			Kind:       "mesh",
			Name:       "mesh-" + c.locality.Datacenter,
			AdminPort:  c.adminPort,
			Ports:      []int{c.proxyPort},
			Logs:       log,
		})
	}, c.gatewayArgs())
}

func (c *ConsulMeshGateway) gatewayArgs() []string {
	return []string{
		"connect", "envoy",
		"-gateway", "mesh",
		"-register",
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", c.adminPort),
		"-http-addr", c.locality.getAddress(),
		"-address", fmt.Sprintf("127.0.0.1:%d", c.proxyPort),
		"--", "-l", "trace",
	}
}
