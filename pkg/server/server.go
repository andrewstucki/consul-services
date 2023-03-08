package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

// Server is a control server for all the services running.
type Server struct {
	// SocketPath is the path to the control socket.
	SocketPath string

	// services contains the registered services
	services []Service
	// mutex guards the service registration
	mutex sync.RWMutex
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
	router.HandleFunc("/services", s.listServices)

	server := http.Server{
		Handler: router,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	defer server.Close()

	listener, err := net.Listen("unix", s.SocketPath)
	if err != nil {
		return err
	}

	go server.Serve(listener)

	<-ctx.Done()
	return nil
}

// Register adds the service to the control server.
func (s *Server) Register(svc Service) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.services = append(s.services, svc)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	w.Header().Set("content-type", "application/json")
	encoder.Encode(s.services)
}
