package server

// Consul contains information about registered Consul instances
type Consul struct {
	Datacenter string
	Ports      []int
	NamedPorts map[string]int
	Logs       string
	Config     string
	Address    string `json:"-"`
	WanAddress string `json:"-"`
}
