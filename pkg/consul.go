package pkg

import (
	"context"
	"errors"

	"github.com/cenkalti/backoff"
	"github.com/hashicorp/consul/api"
)

// ConsulAgent is a Consul agent in dev mode
type ConsulAgent struct {
	*ConsulCommand
}

// Run runs the Consul agent
func (c *ConsulAgent) Run(ctx context.Context) error {
	return c.runConsulBinary(ctx, nil, []string{"agent", "-dev"})
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
