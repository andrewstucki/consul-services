package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Client is a client for the control server.
type Client struct {
	// SocketPath is the path to the control socket.
	SocketPath string

	client *http.Client
}

// NewClient returns an instance of a client.
func NewClient(socket string) *Client {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", socket)
			},
		},
	}

	return &Client{
		SocketPath: socket,
		client:     client,
	}
}

// Get gets a controlled service.
func (c *Client) Get(kind, name string) (*Service, error) {
	url, err := url.Parse(requestPath("/services/" + kind + "/" + name))
	if err != nil {
		return nil, err
	}

	response, err := c.client.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("code: %d, message: %q", response.StatusCode, string(body))
	}

	service := &Service{}
	if err := json.Unmarshal(body, service); err != nil {
		return nil, err
	}
	return service, nil
}

// List lists the controlled services.
func (c *Client) List(kinds ...string) ([]Service, error) {
	url, err := url.Parse(requestPath("/services"))
	if err != nil {
		return nil, err
	}

	if len(kinds) > 0 {
		query := url.Query()
		query.Set("kinds", strings.Join(kinds, ","))
		url.RawQuery = query.Encode()
	}

	response, err := c.client.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("code: %d, message: %q", response.StatusCode, string(body))
	}

	services := []Service{}
	if err := json.Unmarshal(body, &services); err != nil {
		return nil, err
	}
	return services, nil
}

func requestPath(path string) string {
	return fmt.Sprintf("http://unix/%s", strings.TrimPrefix(path, "/"))
}
