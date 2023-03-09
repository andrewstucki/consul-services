package server

// Service is a service running on the mesh.
type Service struct {
	Kind       string
	Name       string
	AdminPort  int
	NamedPorts map[string]int
	Ports      []int
	Logs       string
}
