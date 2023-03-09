package pkg

import (
	"context"
	"fmt"
	"io/fs"
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
	group, ctx := errgroup.WithContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// we want to register all of our services
	// with the control server so we can return
	// information about them
	controlServer := server.New(r.config.Socket)
	group.Go(func() error {
		return controlServer.Run(ctx)
	})

	if r.config.RunConsul {
		consul := &ConsulAgent{
			ConsulCommand: r.config.consulCommand,
		}
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
	}

	upstreams, externalServices := r.initializeExternalServices(controlServer)
	for i := range externalServices {
		service := externalServices[i]

		group.Go(func() error {
			return service.Run(ctx)
		})
	}
	r.waitForNRegistrations(ctx, len(externalServices))

	meshServices := r.initializeMeshServices(controlServer, upstreams)
	for i := range meshServices {
		service := meshServices[i]

		group.Go(func() error {
			return service.Run(ctx)
		})
	}
	r.waitForNRegistrations(ctx, len(meshServices))

	if r.config.ResourceFolder != "" {
		if err := filepath.Walk(r.config.ResourceFolder, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			if !strings.HasSuffix(info.Name(), ".hcl") {
				return nil
			}

			entry, err := parseFileIntoEntry(r.config.consulCommand, path)
			if err != nil {
				return err
			}

			switch e := entry.(type) {
			case *ConsulConfigEntry:
				return e.Write(ctx)
			case *ConsulGateway:
				e.Server = controlServer

				group.Go(func() error {
					return e.Run(ctx)
				})
			}

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

func (r *Runner) initializeExternalServices(server *server.Server) ([]string, []*ConsulExternalService) {
	upstreams := []string{}
	services := []*ConsulExternalService{}

	for i := 1; i <= r.config.ExternalHTTPServiceCount; i++ {
		upstreams = append(upstreams, httpExternalServiceName(i))
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulExternalService{
				ConsulCommand: r.config.consulCommand,
				ID:            httpExternalServiceID(i, j),
				Name:          httpExternalServiceName(i),
				Protocol:      protocolHTTP,
				OnRegister:    r.registrationCh,
				Server:        server,
				tracker:       newTracker(),
			})
		}
	}

	for i := 1; i <= r.config.ExternalTCPServiceCount; i++ {
		upstreams = append(upstreams, tcpExternalServiceName(i))
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulExternalService{
				ConsulCommand: r.config.consulCommand,
				ID:            tcpExternalServiceID(i, j),
				Name:          tcpExternalServiceName(i),
				Protocol:      protocolTCP,
				OnRegister:    r.registrationCh,
				Server:        server,
				tracker:       newTracker(),
			})
		}
	}

	return upstreams, services
}

func (r *Runner) initializeMeshServices(server *server.Server, upstreams []string) []*ConsulMeshService {
	services := []*ConsulMeshService{}

	for i := 1; i <= r.config.HTTPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand:     r.config.consulCommand,
				ID:                httpServiceID(i, j),
				Name:              httpServiceName(i),
				Protocol:          protocolHTTP,
				OnRegister:        r.registrationCh,
				Server:            server,
				ExternalUpstreams: upstreams,
				tracker:           newTracker(),
			})
		}
	}

	for i := 1; i <= r.config.TCPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand:     r.config.consulCommand,
				ID:                tcpServiceID(i, j),
				Name:              tcpServiceName(i),
				Protocol:          protocolTCP,
				OnRegister:        r.registrationCh,
				Server:            server,
				ExternalUpstreams: upstreams,
				tracker:           newTracker(),
			})
		}
	}

	return services
}

func httpServiceID(i, j int) string {
	return fmt.Sprintf("http-%d-%d", i, j)
}

func httpServiceName(i int) string {
	return fmt.Sprintf("http-%d", i)
}

func httpExternalServiceID(i, j int) string {
	return fmt.Sprintf("http-external-%d-%d", i, j)
}

func httpExternalServiceName(i int) string {
	return fmt.Sprintf("http-external-%d", i)
}

func tcpServiceID(i, j int) string {
	return fmt.Sprintf("tcp-%d-%d", i, j)
}

func tcpServiceName(i int) string {
	return fmt.Sprintf("tcp-%d", i)
}

func tcpExternalServiceID(i, j int) string {
	return fmt.Sprintf("tcp-external-%d-%d", i, j)
}

func tcpExternalServiceName(i int) string {
	return fmt.Sprintf("tcp-external-%d", i)
}
