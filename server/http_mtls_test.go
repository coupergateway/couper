package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server"
)

func TestHTTPSServer_TLS_SelfSigned(t *testing.T) {
	helper := test.New(t)

	client := test.NewHTTPSClient(&tls.Config{
		RootCAs: x509.NewCertPool(),
	})

	shutdown, _ := newCouper("testdata/mtls/01_couper.hcl", helper)
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost/", nil)
	helper.Must(err)

	_, err = client.Do(outreq)
	if err == nil {
		t.Fatal("tls error expected, got nil")
	}

	if err.Error() != `Get "https://localhost/": x509: certificate signed by unknown authority` {
		t.Errorf("Want unknown authority error, got: %v", err)
	}
}

func TestHTTPSServer_TLS_ServerCertificate(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs: pool,
	})

	shutdown, _ := newCouperWithTemplate("testdata/mtls/02_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate), // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),  // PEM
	})
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost/", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}
}

func TestHTTPSServer_TLS_ServerClientCertificate(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	shutdown, _ := newCouperWithTemplate("testdata/mtls/03_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
	})
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost/", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}

	// without presenting the client certificate
	client = test.NewHTTPSClient(&tls.Config{
		RootCAs: pool,
	})

	res, err = client.Do(outreq)
	if err == nil {
		t.Fatal("expected a remote tls error")
	}

	if err.Error() != `Get "https://localhost/": remote error: tls: bad certificate` {
		t.Errorf("Expected a tls handshake error, got: %v", err)
	}
}
