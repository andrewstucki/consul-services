package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-hclog"
)

// Server is a control server for all the services running.
type Server struct {
	// SocketPath is the path to the control socket.
	SocketPath string

	// Logger for errors
	Logger hclog.Logger

	// Datacenters are the names of the datacenters this server tracks resources for.
	Datacenters []string

	// consuls contains the registered consul instances
	consuls []Consul
	// services contains the registered services
	services []Service
	// entries contains the registered user-provided config entries
	entries []Entry
	// mutex guards the service registration
	mutex sync.RWMutex
	// server is a handle to the http server
	server *http.Server
}

// New creates a new control server.
func New(logger hclog.Logger, path string, datacenters []string) *Server {
	return &Server{
		SocketPath:  path,
		Logger:      logger,
		Datacenters: datacenters,
	}
}

// Run runs the control server.
func (s *Server) Run(ctx context.Context) error {
	router := mux.NewRouter()
	router.HandleFunc("/shutdown", s.shutdown)
	router.HandleFunc("/services", s.listServices)
	router.HandleFunc("/services/{kind}/{name}", s.getService)
	router.HandleFunc("/consul/{dc}", s.getConsul)
	router.HandleFunc("/report", s.getReport)

	s.server = &http.Server{
		Handler: router,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	defer s.server.Close()

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return err
	}

	errChannel := make(chan error, 1)
	go func() {
		if err := s.server.Serve(listener); err != nil {
			errChannel <- err
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChannel:
		return err
	}
}

// Register adds the service to the control server.
func (s *Server) Register(svc Service) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.services = append(s.services, svc)
}

// AddConsul adds the consul instance to the control server.
func (s *Server) AddConsul(consul Consul) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.consuls = append(s.consuls, consul)
}

// AddEntry adds the entry instance to the control server.
func (s *Server) AddEntry(entry Entry) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.entries = append(s.entries, entry)
}

func (s *Server) shutdown(w http.ResponseWriter, r *http.Request) {
	defer s.server.Shutdown(context.Background())
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	kindsParam := r.URL.Query().Get("kinds")
	kinds := map[string]bool{}
	if kindsParam != "" {
		for _, kind := range strings.Split(kindsParam, ",") {
			kinds[kind] = true
		}
	}

	encoder := json.NewEncoder(w)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	services := []Service{}
	for _, service := range s.services {
		if len(kinds) == 0 || kinds[service.Kind] {
			services = append(services, service)
		}
	}

	sort.SliceStable(services, func(i, j int) bool {
		if services[i].Kind != services[j].Kind {
			return services[i].Kind < services[j].Kind
		}
		return services[i].Name < services[j].Name
	})

	w.Header().Set("content-type", "application/json")
	encoder.Encode(services)
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	kind := params["kind"]
	name := params["name"]

	encoder := json.NewEncoder(w)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, service := range s.services {
		if kind == service.Kind && name == service.Name {
			w.Header().Set("content-type", "application/json")
			encoder.Encode(service)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "not found")
}

func (s *Server) getConsul(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	datacenter := params["dc"]

	encoder := json.NewEncoder(w)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, consul := range s.consuls {
		if datacenter == consul.Datacenter {
			w.Header().Set("content-type", "application/json")
			encoder.Encode(consul)
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "not found")
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	snapshot := s.snapshot()
	operations, err := snapshot.Operations()

	if err != nil {
		s.Logger.Error("snapshot generation error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
		return
	}

	var builder strings.Builder

	_, err = builder.WriteString(scriptHead)
	if err != nil {
		s.Logger.Error("script building error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
		return
	}
	for _, op := range operations {
		script := op.Script()
		if _, ok := op.(Block); !ok {
			script = "\n" + strings.TrimSpace(script) + "\n"
		}

		_, err := builder.WriteString(script)
		if err != nil {
			s.Logger.Error("script building error", "err", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "internal error")
			return
		}
	}
	_, err = builder.WriteString(scriptTail)
	if err != nil {
		s.Logger.Error("script building error", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal error")
		return
	}

	fmt.Fprintf(w, builder.String())
}

var knownGateways = map[string]string{
	api.APIGateway:         "api",
	api.IngressGateway:     "ingress",
	api.TerminatingGateway: "terminating",
}

func (s *Server) snapshot() Snapshot {
	var snapshot Snapshot

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, dc := range s.Datacenters {
		datacenter := DatacenterSnapshot{
			Datacenter: dc,
		}
		for i := range s.consuls {
			consul := s.consuls[i]
			if consul.Datacenter == dc {
				datacenter.Consul = &consul
				break
			}
		}
		for _, service := range s.services {
			if service.Datacenter == dc {
				switch {
				case service.Kind == "service":
					datacenter.Services = append(datacenter.Services, service)
				case service.Kind == "external":
					datacenter.ExternalServices = append(datacenter.ExternalServices, service)
				case service.Kind == "mesh":
					// special case the mesh gateways since they need to be booted up early
					datacenter.MeshGateways = append(datacenter.MeshGateways, service)
				case knownGateways[service.Kind] != "":
					datacenter.Gateways = append(datacenter.Gateways, service)
				default:
					datacenter.ServiceProxies = append(datacenter.ServiceProxies, service)
				}
			}
		}
		for _, entry := range s.entries {
			if entry.Datacenter == dc {
				datacenter.ConfigEntries = append(datacenter.ConfigEntries, entry)
			}
		}
		snapshot.Datacenters = append(snapshot.Datacenters, datacenter)
	}

	snapshot.logger = s.Logger

	return snapshot
}
