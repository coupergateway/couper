package server

import (
	"crypto"
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
	"strconv"
	"sync/atomic"
	"time"
)

type SelfSignedCertificate struct {
	CA                            *tls.Certificate
	CACertificate                 PEM
	Server                        *tls.Certificate
	ServerCertificate             PEM
	ServerPrivateKey              []byte
	Client                        *tls.Certificate
	ClientCertificate             PEM
	ClientPrivateKey              []byte
	ClientIntermediateCertificate PEM
	ClientIntermediate            *tls.Certificate
}

type PEM struct {
	Certificate []byte
	PrivateKey  []byte
}

var caCount uint32 = 1

// NewCertificate creates a certificate with given hosts and duration.
// If no hosts are provided all localhost variants will be used.
func NewCertificate(duration time.Duration, hosts []string, notBefore *time.Time) (*SelfSignedCertificate, error) {
	parentName := "rootCA_" + strconv.Itoa(int(atomic.LoadUint32(&caCount)))
	defer atomic.AddUint32(&caCount, 1)

	rootCA, rootPEM, err := newCertificateAuthority(parentName, "", nil, nil)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	// server
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := defaultTemplate()

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
		return nil, err
	}

	srvCrt, srvKeyBytes, srvPEM, err := newCertificateFromDER(srvDER, privateKey)
	if err != nil {
		return nil, err
	}

	// intermediate
	intermediateName := "intermediateCA_" + strconv.Itoa(int(atomic.LoadUint32(&caCount)))
	interCA, interPEM, err := newCertificateAuthority(intermediateName, parentName, rootCA.Leaf.PublicKey, rootCA.PrivateKey)
	if err != nil {
		return nil, err
	}

	// client
	privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template = defaultTemplate()
	template.NotAfter = notBefore.Add(duration)
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	template.MaxPathLenZero = true

	clientDER, err := x509.CreateCertificate(rand.Reader, &template, interCA.Leaf, &privateKey.PublicKey, interCA.PrivateKey)
	if err != nil {
		return nil, err
	}

	clientCrt, ClientKeyBytes, clientPEM, err := newCertificateFromDER(clientDER, privateKey)

	return &SelfSignedCertificate{
		CA:                            rootCA,
		CACertificate:                 *rootPEM,
		Server:                        srvCrt,
		ServerCertificate:             *srvPEM,
		ServerPrivateKey:              srvKeyBytes,
		Client:                        clientCrt,
		ClientCertificate:             *clientPEM,
		ClientPrivateKey:              ClientKeyBytes,
		ClientIntermediate:            interCA,
		ClientIntermediateCertificate: *interPEM,
	}, err
}

func newCertificateAuthority(name, parentName string, publicKey any, privateKey crypto.PrivateKey) (*tls.Certificate, *PEM, error) {
	pubKey := publicKey
	pathLen := 2
	if _, ok := privateKey.(*ecdsa.PrivateKey); !ok {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		pubKey = &key.PublicKey
		privateKey = key
		pathLen = 1
	}

	template := defaultTemplate()
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	template.NotAfter = template.NotBefore.Add(time.Hour * 24)
	template.Subject = pkix.Name{
		CommonName:         name,
		Country:            template.Subject.Country,
		Organization:       template.Subject.Organization,
		OrganizationalUnit: template.Subject.OrganizationalUnit,
	}
	template.BasicConstraintsValid = true
	template.MaxPathLen = pathLen

	if parentName != "" {
		template.Issuer = template.Subject
		template.Issuer.CommonName = parentName
	}

	caDER, err := x509.CreateCertificate(rand.Reader, &template, &template, pubKey, privateKey)
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

func newSerialNumber() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	i, _ := rand.Int(rand.Reader, serialNumberLimit)
	return i
}

func defaultTemplate() x509.Certificate {
	return x509.Certificate{
		Subject: pkix.Name{
			Country:            []string{"DE"},
			Organization:       []string{"Couper"},
			OrganizationalUnit: []string{"Development"},
		},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Issuer:       pkix.Name{CommonName: "github/avenga/couper/server"},
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		NotBefore:    time.Now(),
		SerialNumber: newSerialNumber(),
	}
}
