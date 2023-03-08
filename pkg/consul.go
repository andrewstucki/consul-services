package pkg

import (
	"bytes"
	"context"
	"errors"
	"os/exec"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/consul/api"
)

// ConsulAgent is a Consul agent in dev mode
type ConsulAgent struct {
	// ConsulBinary is the path on the system to the Consul binary used to invoke registration and connect commands.
	ConsulBinary string
}

// Run runs the Consul agent
func (c *ConsulAgent) Run(ctx context.Context) error {
	var errBuffer bytes.Buffer

	cmd := exec.CommandContext(ctx, c.ConsulBinary, "agent", "-dev")
	cmd.Stderr = &errBuffer

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return errors.New(errBuffer.String())
		}
		return err
	}

	return nil
}

func (c *ConsulAgent) ready(ctx context.Context) error {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	return backoff.Retry(func() error {
		options := &api.QueryOptions{}
		_, meta, err := client.Catalog().Nodes(options.WithContext(ctx))
		if err != nil {
			return err
		}
		if !meta.KnownLeader {
			return errors.New("no known consul leader")
		}
		return nil
	}, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 20), ctx))
}
