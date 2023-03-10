package commands

import "fmt"

func GatewayRegistrationArgs(kind, name, address string, adminPort, registrationPort int) []string {
	return []string{
		"connect", "envoy",
		"-gateway", kind,
		"-register",
		"-service", name,
		"-proxy-id", name,
		"-admin-bind", fmt.Sprintf("127.0.0.1:%d", adminPort),
		"-http-addr", address,
		"-address", fmt.Sprintf("127.0.0.1:%d", registrationPort),
		"--", "-l", "trace",
	}
}
