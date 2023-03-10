package server

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/andrewstucki/consul-services/pkg/commands"
	"github.com/andrewstucki/consul-services/pkg/vfs"
	"github.com/hashicorp/go-hclog"
)

// DatacenterSnapshot is the snapshot of everything in a given datacenter
type DatacenterSnapshot struct {
	Datacenter       string
	Consul           *Consul
	ExternalServices []Service
	ServiceProxies   []Service
	Services         []Service
	MeshGateways     []Service
	Gateways         []Service
	ConfigEntries    []Entry
}

// Snapshot is the snapshot of everything we've created
type Snapshot struct {
	Datacenters []DatacenterSnapshot
	logger      hclog.Logger
}

var (
	scriptHead = `#!/bin/bash
cleanup() {
  echo "shutting down upstreams"
}

trap 'trap " " SIGTERM; kill 0; wait; cleanup' SIGINT SIGTERM

waitFor() {
  local address=$1
  
  local timeout=10
  local interval=2
  
  local start_time end_time
  start_time=$(date +%%s)
  end_time=$((start_time + timeout))

  while [ $(date +%%s) -lt $end_time ]; do
    if curl -s -v $address/v1/catalog/nodes 2>&1 | grep "X-Consul-Knownleader: true"; then
      return 0
    fi
    sleep $interval
  done
  >&2 echo "Timeout exceeded."
  exit 1
}

rm -f consul-services-output.log

`
	scriptTail = `
wait
`
)

type Block string

func (b Block) Script() string {
	content := "# " + strings.TrimSpace(string(b)) + " #"
	border := strings.Repeat("#", len(content))
	return fmt.Sprintf(`
%s
%s
%s
`, border, content, border)
}

// RegisteredFile is a file along with the command to register it to Consul
type RegisteredFile struct {
	Message             string
	RegistrationCommand string
	Name                string
	Data                []byte
}

func (r *RegisteredFile) Script() string {
	return fmt.Sprintf(`echo "%s"
cat << EOE > %s
%s
EOE
%s
`, r.Message, r.Name, strings.TrimSpace(string(r.Data)), r.RegistrationCommand)
}

type Mkdir struct {
	dc string
}

func (c *Mkdir) Script() string {
	return fmt.Sprintf(`mkdir -p %s`, tmpFilename(path.Join(c.dc, "{consul,entries}")))
}

type ConsulWait struct {
	address string
}

func (c *ConsulWait) Script() string {
	return fmt.Sprintf(`waitFor %s`, c.address)
}

type Join struct {
	dc      string
	address string
	wans    []string
}

func (j *Join) Script() string {
	return fmt.Sprintf(`echo "Running join on '%s' Consul"
%s`, j.dc, commands.ConsulCommand(commands.AgentJoinArgs(j.address, j.wans)))
}

type RunGateway struct {
	service Service
}

func (g *RunGateway) Script() string {
	return fmt.Sprintf(`echo "Running '%s' gateway '%s'"
%s`, g.service.Kind, g.service.Name, background(commands.ConsulCommand(commands.GatewayRegistrationArgs(
		strings.TrimSuffix(g.service.Kind, "-gateway"),
		g.service.Name,
		g.service.ConsulAddress,
		g.service.AdminPort,
		g.service.RegisteredPort,
	))))
}

type RunService struct {
	service Service
}

func (s *RunService) Script() string {
	switch s.service.Protocol {
	case "tcp":
		return fmt.Sprintf(`echo "Running 'tcp' service '%s'"
ncat -e "/bin/echo %s" -k -l %d &`, s.service.Name, s.service.Name, s.service.ServicePort)
	case "http":
		return fmt.Sprintf(`echo "Running 'http' service '%s'"
cat << EOF > %s
HTTP/1.1 200 GET
Content-Type: text/html; charset=UTF-8

<!doctype html><html><body>%s</body></html>
EOF
ncat -e "/bin/cat %s" -k -l %d &
`,
			s.service.Name,
			tmpFilename(fmt.Sprintf("%s.html", s.service.Name)),
			s.service.Name,
			tmpFilename(fmt.Sprintf("%s.html", s.service.Name)),
			s.service.ServicePort,
		)
	default:
		return fmt.Sprintf("# unsupported service type %s for service %q", s.service.Protocol, s.service.Name)
	}
}

type RunSidecar struct {
	service Service
}

func (s *RunSidecar) Script() string {
	return fmt.Sprintf(`echo "Running sidecar for '%s'"
%s`, s.service.Name, background(commands.ConsulCommand(commands.SidecarArgs(
		s.service.ConsulAddress,
		s.service.Name,
		s.service.AdminPort,
	))))
}

// OrderedOperation creates an operation that can be transformed into a part of a script.
type OrderedOperation interface {
	Script() string
}

// Operations returns all of the operations needed in order to recreate this run.
func (s *Snapshot) Operations() ([]OrderedOperation, error) {
	var operations []OrderedOperation

	operations = append(operations, Block("Writing Consul Configuration(s)"))
	wans := []string{}
	for _, dc := range s.Datacenters {
		// write the consul configs
		if dc.Consul != nil {
			wans = append(wans, dc.Consul.WanAddress)
			data, err := vfs.ReadFile(dc.Consul.Config)
			if err != nil {
				return nil, err
			}
			operations = append(operations, &Mkdir{
				dc: dc.Datacenter,
			}, &RegisteredFile{
				Message: fmt.Sprintf("Running '%s' Consul", dc.Datacenter),
				RegistrationCommand: background(commands.ConsulCommand(commands.AgentRunArgs(
					tmpFilename(dc.Consul.Config),
				))),
				Name: tmpFilename(dc.Consul.Config),
				Data: data,
			}, &ConsulWait{
				address: dc.Consul.Address,
			})
		}
	}

	if len(s.Datacenters) > 1 {
		operations = append(operations, Block("Joining Consul Clusters"))
		// join the consuls into a federated cluster, in backwards order
		for i := len(s.Datacenters) - 1; i >= 0; i-- {
			dc := s.Datacenters[i]

			if dc.Consul != nil {
				filtered := []string{}
				for _, wan := range wans {
					if wan == dc.Consul.WanAddress {
						continue
					}
					filtered = append(filtered, wan)
				}
				operations = append(operations, &Join{
					dc:      dc.Datacenter,
					address: dc.Consul.Address,
					wans:    filtered,
				})
			}
		}

		operations = append(operations, Block("Starting Mesh Gateways"))

		// start up mesh gateways
		for _, dc := range s.Datacenters {
			for _, mesh := range dc.MeshGateways {
				operations = append(operations, &RunGateway{
					service: mesh,
				})
			}
		}
	}

	// main loop for services
	for _, dc := range s.Datacenters {
		if len(dc.ExternalServices) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Adding %d External Service(s) for %q", len(dc.ExternalServices), dc.Datacenter)))

			// write the external services
			defaults, services, proxies, err := fileSetsForServices(s.logger, dc.ExternalServices, true)
			if err != nil {
				return nil, err
			}
			// append all defaults
			for _, op := range defaults {
				operations = append(operations, op)
			}
			// now services
			for _, op := range services {
				operations = append(operations, op)
			}
			// and finally proxies
			for _, op := range proxies {
				operations = append(operations, op)
			}
		}

		if len(dc.ServiceProxies) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Adding %d Mesh Service(s) for %q", len(dc.ServiceProxies), dc.Datacenter)))

			// now the same the mesh services
			defaults, services, proxies, err := fileSetsForServices(s.logger, dc.ServiceProxies, false)
			if err != nil {
				return nil, err
			}
			// append all defaults
			for _, op := range defaults {
				operations = append(operations, op)
			}
			// now services
			for _, op := range services {
				operations = append(operations, op)
			}
			// and finally proxies
			for _, op := range proxies {
				operations = append(operations, op)
			}
		}

		if len(dc.ConfigEntries) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Adding %d Additional Configuration for %q", len(dc.ConfigEntries), dc.Datacenter)))

			// now all user-specified files
			for _, entry := range dc.ConfigEntries {
				data, err := vfs.ReadFile(entry.File)
				if err != nil {
					return nil, err
				}
				operations = append(operations, &RegisteredFile{
					Message: fmt.Sprintf("Writing Configuration Entry '%s'", entry.File),
					RegistrationCommand: commands.ConsulCommand(commands.WriteConfigArgs(
						entry.Datacenter,
						entry.ConsulAddress,
						tmpFilename(entry.File),
					)),
					Name: tmpFilename(entry.File),
					Data: data,
				})
			}
		}
	}

	// finally boot up all of the services
	for _, dc := range s.Datacenters {
		if len(dc.ExternalServices) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Starting %d External Service(s) for %q", len(dc.ExternalServices), dc.Datacenter)))

			// first external services
			for _, external := range dc.ExternalServices {
				operations = append(operations, &RunService{
					service: external,
				}, &RunSidecar{
					service: external,
				})
			}
		}

		if len(dc.ServiceProxies) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Starting %d Mesh Service(s) for %q", len(dc.ServiceProxies), dc.Datacenter)))

			// then regular
			for _, service := range dc.ServiceProxies {
				service := service
				service.Name = strings.TrimSuffix(service.Name, "-proxy")
				operations = append(operations, &RunService{
					service: service,
				}, &RunSidecar{
					service: service,
				})
			}
		}

		if len(dc.Gateways) > 0 {
			operations = append(operations, Block(fmt.Sprintf("Starting %d Gateway(s) for %q", len(dc.Gateways), dc.Datacenter)))

			// and finally gateways
			for _, gateway := range dc.Gateways {
				operations = append(operations, &RunGateway{
					service: gateway,
				})
			}
		}
	}

	return operations, nil
}

func fileSetsForServices(logger hclog.Logger, services []Service, isExternal bool) (defaults, registrations, proxies []*RegisteredFile, err error) {
	serviceDefaults := []*RegisteredFile{}
	serviceRegistrations := []*RegisteredFile{}
	serviceProxies := []*RegisteredFile{}
	for _, service := range services {
		defaults, registration, proxy, err := fileTupleForService(logger, service, isExternal)
		if err != nil {
			return nil, nil, nil, err
		}
		serviceDefaults = append(serviceDefaults, defaults)
		serviceRegistrations = append(serviceRegistrations, registration)
		serviceProxies = append(serviceProxies, proxy)
	}
	return serviceDefaults, serviceRegistrations, proxies, nil
}

func fileTupleForService(logger hclog.Logger, service Service, isExternal bool) (defaults, registration, proxy *RegisteredFile, err error) {
	name := strings.TrimSuffix(service.Name, "-proxy")

	var defaultBytes, registrationBytes, proxyBytes []byte

	defaultBytes, err = vfs.ReadFile(service.ServiceDefaultsFile)
	if err != nil {
		return
	}
	registrationBytes, err = vfs.ReadFile(service.ServiceRegistrationFile)
	if err != nil {
		return
	}

	if !isExternal {
		proxyBytes, err = vfs.ReadFile(service.ServiceProxyFile)
		if err != nil {
			return
		}
		proxy = &RegisteredFile{
			Message: fmt.Sprintf("Writing Service Proxy Registration for '%s'", name),
			RegistrationCommand: commands.ConsulCommand(commands.RegisterServiceArgs(
				service.Datacenter,
				service.ConsulAddress,
				tmpFilename(service.ServiceProxyFile),
			)),
			Name: tmpFilename(service.ServiceProxyFile),
			Data: proxyBytes,
		}
	}

	defaults = &RegisteredFile{
		Message: fmt.Sprintf("Writing Service Defaults for '%s'", name),
		RegistrationCommand: commands.ConsulCommand(commands.WriteConfigArgs(
			service.Datacenter,
			service.ConsulAddress,
			tmpFilename(service.ServiceDefaultsFile),
		)),
		Name: tmpFilename(service.ServiceDefaultsFile),
		Data: defaultBytes,
	}

	if isExternal {
		registration = &RegisteredFile{
			Message: fmt.Sprintf("Writing Service Registration for '%s'", name),
			RegistrationCommand: fmt.Sprintf("curl --request PUT --data @%s %s/v1/catalog/register",
				tmpFilename(service.ServiceRegistrationFile),
				service.ConsulAddress,
			),
			Name: tmpFilename(service.ServiceRegistrationFile),
			Data: registrationBytes,
		}

		return
	}

	registration = &RegisteredFile{
		Message: fmt.Sprintf("Writing Service Registration for '%s'", name),
		RegistrationCommand: commands.ConsulCommand(commands.RegisterServiceArgs(
			service.Datacenter,
			service.ConsulAddress,
			tmpFilename(service.ServiceRegistrationFile),
		)),
		Name: tmpFilename(service.ServiceRegistrationFile),
		Data: registrationBytes,
	}

	return
}

func tmpFilename(name string) string {
	return path.Join(string(os.PathSeparator), "tmp", name)
}

func background(command string) string {
	return command + " >> consul-services-output.log 2>&1 &"
}
