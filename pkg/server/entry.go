package server

// Entry is a config entry registered with the server.
type Entry struct {
	Datacenter    string
	Partition     string
	Namespace     string
	Kind          string
	Name          string
	File          string
	ConsulAddress string
}
