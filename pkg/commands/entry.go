package commands

func WriteConfigArgs(datacenter, address, path string) []string {
	return []string{
		"config", "write",
		"-datacenter", datacenter,
		"-http-addr", address,
		path,
	}
}
