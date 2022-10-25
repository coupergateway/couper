package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"time"
)

type SelfSignedCertificate struct {
	CA     []byte // PEM encoded
	Server *tls.Certificate
}

// NewCertificate creates a certificate with given hosts and duration.
// If no hosts are provided all localhost variants will be used.
func NewCertificate(duration time.Duration, hosts []string, notBefore *time.Time) (*SelfSignedCertificate, error) {
	caPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1", "::1", "localhost", "0.0.0.0", "::0"}
	}

	if notBefore == nil {
		n := time.Now()
		notBefore = &n
	}
	notAfter := notBefore.Add(duration)

	serialNumber, err := newSerialNumber()
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"DE"},
			Organization:       []string{"Couper"},
			OrganizationalUnit: []string{"Development"},
		},
		NotBefore: *notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// self CA
	caTemplate := template
	caTemplate.SerialNumber = serialNumber
	caTemplate.IsCA = true
	caTemplate.KeyUsage |= x509.KeyUsageCertSign

	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	caCert := &bytes.Buffer{}
	err = pem.Encode(caCert, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	if err != nil {
		return nil, err
	}

	caKey := &bytes.Buffer{}
	err = pem.Encode(caKey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivateKey)})
	if err != nil {
		return nil, err
	}

	// server certificate
	srvPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	srvTemplate := template
	srvTemplate.SerialNumber, err = newSerialNumber()
	if err != nil {
		return nil, err
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			srvTemplate.IPAddresses = append(srvTemplate.IPAddresses, ip)
		} else {
			srvTemplate.DNSNames = append(srvTemplate.DNSNames, h)
		}
	}

	srvDER, err := x509.CreateCertificate(rand.Reader, &srvTemplate, &caTemplate, &srvPrivateKey.PublicKey, caPrivateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	srvCert := &bytes.Buffer{}
	err = pem.Encode(srvCert, &pem.Block{Type: "CERTIFICATE", Bytes: srvDER})
	if err != nil {
		return nil, err
	}

	srvKey := &bytes.Buffer{}
	err = pem.Encode(srvKey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(srvPrivateKey)})
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(srvCert.Bytes(), srvKey.Bytes())
	if err != nil {
		return nil, err
	}
	cert.Leaf, err = x509.ParseCertificate(srvDER)
	return &SelfSignedCertificate{
		CA:     caCert.Bytes(),
		Server: &cert,
	}, err
}

func newSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
