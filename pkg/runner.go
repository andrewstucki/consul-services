package pkg

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

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

	if r.config.RunConsul {
		consul := &ConsulAgent{
			ConsulCommand: r.config.consulCommand,
		}
		group.Go(func() error {
			return consul.Run(ctx)
		})

		if err := consul.ready(ctx); err != nil {
			return err
		}
	}

	services := []*ConsulMeshService{}

	for i := 1; i <= r.config.HTTPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand: r.config.consulCommand,
				ID:            httpServiceID(i, j),
				Name:          httpServiceName(i),
				Protocol:      protocolHTTP,
				OnRegister:    r.registrationCh,
			})
		}
	}

	for i := 1; i <= r.config.TCPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services = append(services, &ConsulMeshService{
				ConsulCommand: r.config.consulCommand,
				ID:            tcpServiceID(i, j),
				Name:          tcpServiceName(i),
				Protocol:      protocolTCP,
				OnRegister:    r.registrationCh,
			})
		}
	}

	for i := range services {
		service := services[i]

		group.Go(func() error {
			return service.Run(ctx)
		})
	}

	serviceCount := len(services)

REGISTRATION_LOOP:
	for {
		select {
		case <-ctx.Done():
			// just let the rest of the cancelation
			// code handle the below blocks
			break REGISTRATION_LOOP
		case <-r.registrationCh:
			serviceCount--
			if serviceCount == 0 {
				break REGISTRATION_LOOP
			}
		}
	}

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
				group.Go(func() error {
					return e.Run(ctx)
				})
			}

			return nil
		}); err != nil {
			return err
		}
	}

	return group.Wait()
}

func httpServiceID(i, j int) string {
	return fmt.Sprintf("http-%d-%d", i, j)
}

func httpServiceName(i int) string {
	return fmt.Sprintf("http-%d", i)
}

func tcpServiceID(i, j int) string {
	return fmt.Sprintf("tcp-%d-%d", i, j)
}

func tcpServiceName(i int) string {
	return fmt.Sprintf("tcp-%d", i)
}
