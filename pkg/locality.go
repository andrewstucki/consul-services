package pkg

import "github.com/hashicorp/consul/api"

type locality struct {
	Datacenter string
	Partition  string
	Namespace  string

	// Consul connection info
	client  *api.Client
	address string
}

func (l locality) getClient() (*api.Client, error) {
	if l.client != nil {
		return l.client, nil
	}

	return api.NewClient(api.DefaultConfig())
}

func (l locality) getAddress() string {
	if l.address != "" {
		return l.address
	}

	return api.DefaultConfig().Address
}
