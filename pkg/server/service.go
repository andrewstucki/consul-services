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
}
