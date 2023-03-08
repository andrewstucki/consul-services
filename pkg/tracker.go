package pkg

type tracker struct {
	ports []int
}

func (t *tracker) GetPort() (int, error) {
	port, err := freePort()
	if err != nil {
		return 0, err
	}

	t.ports = append(t.ports, port)

	return port, nil
}

func (t *tracker) GetCertificate(name string, sans ...string) (*CertificateInfo, error) {
	certificate, err := generateCertificate(name, sans...)
	if err != nil {
		return nil, err
	}

	return certificate, nil
}
