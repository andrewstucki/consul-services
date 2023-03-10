package commands

import (
	"fmt"
)

func RegisterServiceArgs(datacenter, address, file string) []string {
	return []string{
		"services", "register",
		"-datacenter", datacenter,
		"-http-addr", address,
		file,
	}
}

func SidecarArgs(address, id string, adminPort int) []string {
	return []string{
		"connect", "envoy",
		"-http-addr", address,
		"-sidecar-for", id,
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", adminPort),
		"--", "-l", "trace",
	}
}
