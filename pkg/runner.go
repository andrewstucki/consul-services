package pkg

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	folder, err := os.MkdirTemp("", "consul-services-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(folder)

	if r.config.ResourceFolder != "" {
		return r.runFolderBasedServices(ctx, folder)
	}

	return r.runServices(ctx, folder)
}

func (r *Runner) runFolderBasedServices(ctx context.Context, folder string) error {
	group, ctx := errgroup.WithContext(ctx)

	if r.config.RunConsul {
		consul := &ConsulAgent{
			ConsulBinary: r.config.consulBinary,
		}
		group.Go(func() error {
			return consul.Run(ctx)
		})

		if err := consul.ready(ctx); err != nil {
			return err
		}
	}

	if err := r.writeEntries(ctx); err != nil {
		return err
	}

	// run other services here

	if r.config.GatewayFile != "" {
		gateway, err := r.createGateway(folder)
		if err != nil {
			return err
		}

		group.Go(func() error {
			return gateway.Run(ctx)
		})
	}

	<-ctx.Done()
	return nil
}

func (r *Runner) createGateway(folder string) (*ConsulGateway, error) {
	return createGateway(r.config.consulBinary, r.config.GatewayFile, folder, r.config.Logger)
}

func (r *Runner) writeEntries(ctx context.Context) error {
	if r.config.ExtraFilesFolder != "" {
		if err := filepath.Walk(r.config.ExtraFilesFolder, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			config := &ConsulConfigEntry{
				ConsulBinary:   r.config.consulBinary,
				DefinitionFile: path,
				Logger:         r.config.Logger,
			}

			return config.Write(ctx)
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) runServices(ctx context.Context, folder string) error {
	group, ctx := errgroup.WithContext(ctx)

	if r.config.RunConsul {
		consul := &ConsulAgent{
			ConsulBinary: r.config.consulBinary,
		}
		group.Go(func() error {
			return consul.Run(ctx)
		})

		if err := consul.ready(ctx); err != nil {
			return err
		}
	}

	if err := r.writeEntries(ctx); err != nil {
		return err
	}

	services := 0

	for i := 1; i <= r.config.HTTPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services++
			group.Go(runService(ctx, folder, httpServiceID(i, j), httpServiceName(i), r.runHTTPService))
		}
	}

	for i := 1; i <= r.config.TCPServiceCount; i++ {
		for j := 1; j <= r.config.ServiceDuplicates; j++ {
			services++
			group.Go(runService(ctx, folder, tcpServiceID(i, j), tcpServiceName(i), r.runTCPService))
		}
	}

REGISTRATION_LOOP:
	for {
		select {
		case <-ctx.Done():
			// just let the rest of the cancelation
			// code handle the below blocks
			break REGISTRATION_LOOP
		case <-r.registrationCh:
			services--
			if services == 0 {
				break REGISTRATION_LOOP
			}
		}
	}

	if r.config.GatewayFile != "" {
		gateway, err := r.createGateway(folder)
		if err != nil {
			return err
		}

		group.Go(func() error {
			return gateway.Run(ctx)
		})
	}

	return group.Wait()
}

func (r *Runner) runHTTPService(ctx context.Context, folder, id, name string) error {
	r.config.Logger.Info("running HTTP service", "id", id, "name", name)
	defer r.config.Logger.Info("stopping HTTP service", "id", id, "name", name)

	service := &ConsulMeshService{
		ConsulBinary: r.config.consulBinary,
		ID:           id,
		Name:         name,
		Protocol:     protocolHTTP,
		Folder:       folder,
		Logger:       r.config.Logger.With("name", name, "id", id),
		OnRegister:   r.registrationCh,
	}

	return service.Run(ctx)
}

func (r *Runner) runTCPService(ctx context.Context, folder, id, name string) error {
	r.config.Logger.Info("running TCP service", "id", id, "name", name)
	defer r.config.Logger.Info("stopping TCP service", "id", id, "name", name)

	service := &ConsulMeshService{
		ConsulBinary: r.config.consulBinary,
		ID:           id,
		Name:         name,
		Protocol:     protocolTCP,
		Folder:       folder,
		Logger:       r.config.Logger.With("name", name, "id", id),
		OnRegister:   r.registrationCh,
	}

	return service.Run(ctx)
}

type serviceRunner func(ctx context.Context, folder, id, name string) error

func runService(ctx context.Context, folder, id, name string, runner serviceRunner) func() error {
	return func() error {
		return runner(ctx, folder, id, name)
	}
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
