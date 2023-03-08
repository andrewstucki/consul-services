package pkg

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"time"
)

var (
	CA *CertificateInfo
)

func init() {
	rootCA, err := generateSignedCertificate(generateCertificateOptions{
		IsCA: true,
		Name: "RootCA",
	})
	if err != nil {
		panic(err)
	}
	CA = rootCA
}

// CertificateInfo wraps all of the information needed to describe a generated
// certificate
type CertificateInfo struct {
	Certificate string
	PrivateKey  string
	privateKey  *rsa.PrivateKey
	cert        *x509.Certificate
}

type generateCertificateOptions struct {
	CA        *CertificateInfo
	IsCA      bool
	Name      string
	ExtraSANs []string
	Bits      int
}

func generateCertificate(name string, sans ...string) (*CertificateInfo, error) {
	return generateSignedCertificate(generateCertificateOptions{
		CA:        CA,
		Name:      name,
		ExtraSANs: sans,
	})
}

func generateSignedCertificate(options generateCertificateOptions) (*CertificateInfo, error) {
	bits := options.Bits
	if bits == 0 {
		bits = 2048
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	usage := x509.KeyUsageDigitalSignature
	if options.IsCA {
		usage = x509.KeyUsageCertSign
	}

	sans := []string{}
	ips := []net.IP{}

	for _, extra := range options.ExtraSANs {
		addr := net.ParseIP(extra)
		if addr != nil {
			ips = append(ips, addr)
		} else {
			sans = append(sans, extra)
		}
	}

	expiration := time.Now().AddDate(10, 0, 0)
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     sans,
		Subject: pkix.Name{
			Organization:  []string{"Testing, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Fake Street"},
			PostalCode:    []string{"11111"},
			CommonName:    options.Name,
		},
		IsCA:                  options.IsCA,
		IPAddresses:           ips,
		NotBefore:             time.Now().Add(-10 * time.Minute),
		NotAfter:              expiration,
		SubjectKeyId:          []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              usage,
		BasicConstraintsValid: true,
	}
	caCert := cert
	if options.CA != nil {
		caCert = options.CA.cert
	}
	caPrivateKey := privateKey
	if options.CA != nil {
		caPrivateKey = options.CA.privateKey
	}
	data, err := x509.CreateCertificate(rand.Reader, cert, caCert, &privateKey.PublicKey, caPrivateKey)
	if err != nil {
		return nil, err
	}

	certificatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: data,
	})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return &CertificateInfo{
		Certificate: string(certificatePEM),
		PrivateKey:  string(privateKeyPEM),
		cert:        cert,
		privateKey:  privateKey,
	}, nil
}
