package pkg

import "fmt"

type tracker struct {
	ports      []int
	namedPorts map[string]int
}

func newTracker() *tracker {
	return &tracker{
		namedPorts: make(map[string]int),
	}
}

func (t *tracker) GetPort() (int, error) {
	port, err := freePort()
	if err != nil {
		return 0, err
	}

	t.ports = append(t.ports, port)

	return port, nil
}

func (t *tracker) GetNamedPort(name string) (int, error) {
	if _, ok := t.namedPorts[name]; ok {
		return 0, fmt.Errorf("name %q already in-use", name)
	}

	port, err := freePort()
	if err != nil {
		return 0, err
	}

	t.ports = append(t.ports, port)
	t.namedPorts[name] = port

	return port, nil
}

func (t *tracker) GetCertificate(name string, sans ...string) (*CertificateInfo, error) {
	certificate, err := generateCertificate(name, sans...)
	if err != nil {
		return nil, err
	}

	return certificate, nil
}
