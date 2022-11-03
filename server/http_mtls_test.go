package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strings"
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

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
	helper.Must(err)

	_, err = client.Do(outreq)
	if err == nil {
		t.Fatal("tls error expected, got nil")
	}

	if err.Error() != `Get "https://localhost:4443/": x509: certificate signed by unknown authority` {
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

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
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

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
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

	if err.Error() != `Get "https://localhost:4443/": remote error: tls: bad certificate` {
		t.Errorf("Expected a tls handshake error, got: %v", err)
	}
}

func TestHTTPSServer_TLS_ServerClientCertificateLeaf(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	shutdown, _ := newCouperWithTemplate("testdata/mtls/04_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.ClientCertificate.Certificate),             // PEM
	})
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}
}

func TestHTTPSServer_TLS_ServerClientCertificateLeafNoMatch(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	shutdown, hook := newCouperWithTemplate("testdata/mtls/04_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.CACertificate.Certificate),                 // PEM / just a non-matching one
	})
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
	helper.Must(err)

	hook.Reset()

	_, err = client.Do(outreq)
	if err == nil {
		t.Error("expected a tls handshake error")
	}

	entries := hook.AllEntries()
	if len(entries) == 0 {
		t.Fatal("expected log entries")
	}

	if !strings.HasSuffix(entries[0].Message, "tls: client leaf certificate mismatch") {
		t.Errorf("expected leaf mismatch err, got: %v", entries[0].Message)
	}
}

func TestHTTPSServer_TLS_ServerBackendClient(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	shutdown, _ := newCouperWithTemplate("testdata/mtls/05_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.ClientCertificate.Certificate),             // PEM
		"clientKey":  string(selfSigned.ClientCertificate.PrivateKey),              // PEM
		"rootCA":     string(selfSigned.CACertificate.Certificate),                 // PEM
	})
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/inception", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}
}
