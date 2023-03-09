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
)

// Server is a control server for all the services running.
type Server struct {
	// SocketPath is the path to the control socket.
	SocketPath string

	// consuls contains the registered consul instances
	consuls []Consul
	// services contains the registered services
	services []Service
	// mutex guards the service registration
	mutex sync.RWMutex
	// server is a handle to the http server
	server *http.Server
}

// New creates a new control server.
func New(path string) *Server {
	return &Server{
		SocketPath: path,
	}
}

// Run runs the control server.
func (s *Server) Run(ctx context.Context) error {
	router := mux.NewRouter()
	router.HandleFunc("/shutdown", s.shutdown)
	router.HandleFunc("/services", s.listServices)
	router.HandleFunc("/services/{kind}/{name}", s.getService)
	router.HandleFunc("/consul/{dc}", s.getConsul)

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
