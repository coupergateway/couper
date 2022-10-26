package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
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
	CA                *tls.Certificate
	CACertificate     PEM
	Server            *tls.Certificate
	ServerCertificate PEM
	ServerPrivateKey  []byte
}

type PEM struct {
	Certificate []byte
	PrivateKey  []byte
}

// NewCertificate creates a certificate with given hosts and duration.
// If no hosts are provided all localhost variants will be used.
func NewCertificate(duration time.Duration, hosts []string, notBefore *time.Time) (*SelfSignedCertificate, error) {
	rootCA, rootPEM, err := newCertificateAuthority()
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := defaultTemplate()
	template.SerialNumber, err = newSerialNumber()
	if err != nil {
		return nil, err
	}

	if notBefore == nil {
		n := time.Now()
		notBefore = &n
	}
	template.NotAfter = notBefore.Add(duration)

	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1", "::1", "localhost", "0.0.0.0", "::0"}
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	srvDER, err := x509.CreateCertificate(rand.Reader, &template, rootCA.Leaf, &privateKey.PublicKey, rootCA.PrivateKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	srvCrt, keyBytes, srvPEM, err := newCertificateFromDER(srvDER, privateKey)
	return &SelfSignedCertificate{
		CA:                rootCA,
		CACertificate:     *rootPEM,
		Server:            srvCrt,
		ServerCertificate: *srvPEM,
		ServerPrivateKey:  keyBytes,
	}, err
}

func newCertificateAuthority() (*tls.Certificate, *PEM, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	template := defaultTemplate()
	template.SerialNumber, err = newSerialNumber()
	if err != nil {
		return nil, nil, err
	}
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	template.NotAfter = template.NotBefore.Add(time.Hour * 24)

	caDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}
	cert, _, certPEM, err := newCertificateFromDER(caDER, privateKey)
	return cert, certPEM, err
}

func newCertificateFromDER(caDER []byte, key any) (*tls.Certificate, []byte, *PEM, error) {
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	certificate, err := tls.X509KeyPair(caPEM, keyPEM)
	if err != nil {
		return nil, nil, nil, err
	}
	certificate.Leaf, err = x509.ParseCertificate(caDER)
	return &certificate, privBytes, &PEM{caPEM, keyPEM}, err
}

func newSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}

func defaultTemplate() x509.Certificate {
	return x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"DE"},
			Organization:       []string{"Couper"},
			OrganizationalUnit: []string{"Development"},
		},
		NotBefore:             time.Now(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
}
