package pkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// Service is a service that runs on the service mesh
type Service struct {
	ID       string
	Protocol string
	Port     int
}

func (s *Service) Run(ctx context.Context) error {
	switch s.Protocol {
	case protocolHTTP:
		return s.runHTTPService(ctx)
	case protocolTCP:
		return s.runTCPService(ctx)
	default:
		return errors.New("unsupported protocol")
	}
}

func (s *Service) runTCPService(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", s.Port))
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			fmt.Fprintf(conn, s.ID)
			conn.Close()
		}
	}()

	<-ctx.Done()

	return nil
}

func (s *Service) runHTTPService(ctx context.Context) error {
	server := &http.Server{
		Addr: fmt.Sprintf("127.0.0.1:%d", s.Port),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, s.ID)
		}),
	}
	defer server.Close()

	go server.ListenAndServe()

	<-ctx.Done()

	return nil
}
