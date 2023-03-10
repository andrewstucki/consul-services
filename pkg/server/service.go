package server

// Service is a service running on the mesh.
type Service struct {
	Datacenter string
	Partition  string
	Namespace  string
	Kind       string
	Name       string
	AdminPort  int
	NamedPorts map[string]int
	Ports      []int
	Logs       string
	// the below values are all with regard to the registration
	// information of the service
	ServiceDefaultsFile     string `json:"-"`
	ServiceRegistrationFile string `json:"-"`
	ServiceProxyFile        string `json:"-"`
	ConsulAddress           string `json:"-"`
	// for gateways
	RegisteredPort int `json:"-"`
	// for services
	Protocol    string `json:"-"`
	ServicePort int    `json:"-"`
}
