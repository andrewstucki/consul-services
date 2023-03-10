package commands

import (
	"fmt"
	"strings"
)

func AgentRunArgs(config string) []string {
	return []string{
		"agent", "-dev",
		"-config-file", config,
	}
}

func ConsulCommand(args []string) string {
	return fmt.Sprintf("consul %s", strings.Join(args, " "))
}

func AgentJoinArgs(address string, addresses []string) []string {
	return append([]string{
		"join",
		"-http-addr", address,
		"-wan",
	}, addresses...)
}
