package server

// Service is a service running on the mesh.
type Service struct {
	Kind  string
	Name  string
	Ports []int
	Logs  string
}
