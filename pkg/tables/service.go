package tables

import (
	"io"
	"strconv"
	"strings"

	"github.com/andrewstucki/consul-services/pkg/server"
	"github.com/olekukonko/tablewriter"
)

// PrintServices pretty prints services in a table
func PrintServices(w io.Writer, services []server.Service) {
	var serviceTable [][]string
	for _, service := range services {
		serviceTable = append(serviceTable, formatService(service))
	}

	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Kind", "Name", "Ports"})
	table.SetRowLine(true)
	table.AppendBulk(serviceTable)

	table.Render()
}

func formatService(service server.Service) []string {
	ports := []string{}
	for _, port := range service.Ports {
		ports = append(ports, strconv.Itoa(port))
	}
	return []string{service.Kind, service.Name, strings.Join(ports, ", ")}
}
