package pkg

import (
	"context"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/andrewstucki/consul-services/pkg/server"
	"golang.org/x/sync/errgroup"
)

// Runner runs the configured Consul test services on a mesh.
type Runner struct {
	config         RunnerConfig
	registrationCh chan struct{}
}

// NewRunner creates a new test service runner.
func NewRunner(config RunnerConfig) *Runner {
	return &Runner{
		config:         config,
		registrationCh: make(chan struct{}),
	}
}

// Run runs the desired test services.
func (r *Runner) Run(ctx context.Context) error {
	r.config.Logger.Info("starting runtime")

	group, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// we want to register all of our services
	// with the control server so we can return
	// information about them
	controlServer := server.New(r.config.Logger, r.config.Socket, r.config.Datacenters)
	group.Go(func() error {
		return controlServer.Run(ctx)
	})

	agents := []*ConsulAgent{}
	addresses := []string{}

	meshGatewayServices := []*ConsulMeshGateway{}
	externalServices := []*ConsulExternalService{}
	meshServices := []*ConsulMeshService{}
	resources := []interface{}{}

	for _, dc := range r.config.Datacenters {
		locale := locality{
			Datacenter: dc,
			// Add namespace/partition support later
		}

		if r.config.RunConsul {
			consul := &ConsulAgent{
				ConsulCommand:     r.config.consulCommand,
				Server:            controlServer,
				Datacenter:        dc,
				PrimaryDatacenter: r.config.Datacenters[0],
				tracker:           newTracker(),
			}

			if err := consul.Write(); err != nil {
				return err
			}

			agents = append(agents, consul)
			addresses = append(addresses, consul.wanAddress())

			group.Go(func() error {
				return consul.Run(ctx)
			})

			if err := consul.ready(ctx); err != nil {
				select {
				case <-ctx.Done():
					return group.Wait()
				default:
					return err
				}
			}

			client, err := consul.client()
			if err != nil {
				return err
			}
			locale.client = client
			locale.address = consul.address()
		}

		// register mesh gateway
		meshGatewayServices = append(meshGatewayServices, &ConsulMeshGateway{
			ConsulCommand: r.config.consulCommand,
			Server:        controlServer,
			locality:      locale,
		})

		upstreams, external := r.initializeExternalServices(locale, controlServer)
		externalServices = append(externalServices, external...)

		services := r.initializeMeshServices(locale, controlServer, upstreams)
		meshServices = append(meshServices, services...)

		if r.config.ResourceFolder != "" {
			folder := r.config.ResourceFolder
			if len(r.config.Datacenters) > 1 {
				folder = path.Join(folder, dc)
			}

			if err := filepath.Walk(folder, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				if !strings.HasSuffix(info.Name(), ".hcl") {
					return nil
				}

				entry, err := parseFileIntoEntry(controlServer, r.config.consulCommand, path, locale)
				if err != nil {
					return err
				}

				resources = append(resources, entry)

				return nil
			}); err != nil {
				select {
				case <-ctx.Done():
					return group.Wait()
				default:
					return err
				}
			}
		}
	}

	// reverse the order of the join so that the first
	// listed DC winds up being the primary
	for i := len(agents) - 1; i >= 0; i-- {
		agent := agents[i]
		if err := agent.join(ctx, addresses); err != nil {
			select {
			case <-ctx.Done():
				return group.Wait()
			default:
				return err
			}
		}
	}

	// now register all of the stuff since we're federated

	for i := range meshGatewayServices {
		mesh := meshGatewayServices[i]
		group.Go(func() error {
			return mesh.Run(ctx)
		})
	}

	for i := range externalServices {
		service := externalServices[i]
		group.Go(func() error {
			return service.Run(ctx)
		})
	}
	r.waitForNRegistrations(ctx, len(externalServices))

	for i := range meshServices {
		service := meshServices[i]
		group.Go(func() error {
			return service.Run(ctx)
		})
	}
	r.waitForNRegistrations(ctx, len(meshServices))

	for _, entry := range resources {
		switch e := entry.(type) {
		case *ConsulConfigEntry:
			if err := e.Write(ctx); err != nil {
				select {
				case <-ctx.Done():
					return group.Wait()
				default:
					return err
				}
			}
		case *ConsulGateway:
			e.Server = controlServer

			group.Go(func() error {
				return e.Run(ctx)
			})
		}
	}

	return group.Wait()
}

func (r *Runner) waitForNRegistrations(ctx context.Context, n int) {
	if n <= 0 {
		return
	}

	for {
		select {
		case <-ctx.Done():
			// just let the rest of the cancellation fall through
			return
		case <-r.registrationCh:
			n--
			if n == 0 {
				return
			}
		}
	}
}

func (r *Runner) initializeExternalServices(locality locality, server *server.Server) ([]string, []*ConsulExternalService) {
	upstreams := []string{}
	services := []*ConsulExternalService{}

	for i := 1; i <= r.config.ExternalHTTPServiceCount; i++ {
		upstreams = append(upstreams, httpExternalServiceName(i))
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulExternalService{
				ConsulCommand: r.config.consulCommand,
				ID:            httpExternalServiceID(locality, i, j),
				Name:          httpExternalServiceName(i),
				Protocol:      protocolHTTP,
				OnRegister:    r.registrationCh,
				Server:        server,
				tracker:       newTracker(),
				locality:      locality,
			})
		}
	}

	for i := 1; i <= r.config.ExternalTCPServiceCount; i++ {
		upstreams = append(upstreams, tcpExternalServiceName(i))
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulExternalService{
				ConsulCommand: r.config.consulCommand,
				ID:            tcpExternalServiceID(locality, i, j),
				Name:          tcpExternalServiceName(i),
				Protocol:      protocolTCP,
				OnRegister:    r.registrationCh,
				Server:        server,
				tracker:       newTracker(),
				locality:      locality,
			})
		}
	}

	return upstreams, services
}

func (r *Runner) initializeMeshServices(locality locality, server *server.Server, upstreams []string) []*ConsulMeshService {
	services := []*ConsulMeshService{}

	for i := 1; i <= r.config.HTTPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand:     r.config.consulCommand,
				ID:                httpServiceID(locality, i, j),
				Name:              httpServiceName(i),
				Protocol:          protocolHTTP,
				OnRegister:        r.registrationCh,
				Server:            server,
				ExternalUpstreams: upstreams,
				tracker:           newTracker(),
				locality:          locality,
			})
		}
	}

	for i := 1; i <= r.config.TCPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand:     r.config.consulCommand,
				ID:                tcpServiceID(locality, i, j),
				Name:              tcpServiceName(i),
				Protocol:          protocolTCP,
				OnRegister:        r.registrationCh,
				Server:            server,
				ExternalUpstreams: upstreams,
				tracker:           newTracker(),
				locality:          locality,
			})
		}
	}

	return services
}

func httpServiceID(locality locality, i, j int) string {
	return fmt.Sprintf("http-%s-%d-%d", localitySuffix(locality), i, j)
}

func httpServiceName(i int) string {
	return fmt.Sprintf("http-%d", i)
}

func httpExternalServiceID(locality locality, i, j int) string {
	return fmt.Sprintf("http-external-%s-%d-%d", localitySuffix(locality), i, j)
}

func httpExternalServiceName(i int) string {
	return fmt.Sprintf("http-external-%d", i)
}

func tcpServiceID(locality locality, i, j int) string {
	return fmt.Sprintf("tcp-%s-%d-%d", localitySuffix(locality), i, j)
}

func tcpServiceName(i int) string {
	return fmt.Sprintf("tcp-%d", i)
}

func tcpExternalServiceID(locality locality, i, j int) string {
	return fmt.Sprintf("tcp-external-%s-%d-%d", localitySuffix(locality), i, j)
}

func tcpExternalServiceName(i int) string {
	return fmt.Sprintf("tcp-external-%d", i)
}

func localitySuffix(locality locality) string {
	suffix := locality.Datacenter
	if locality.Partition != "" {
		suffix += "-" + locality.Partition
	}
	if locality.Namespace != "" {
		suffix += "-" + locality.Namespace
	}
	return suffix
}
