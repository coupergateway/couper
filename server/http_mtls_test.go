package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server"
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

	if err.Error() != `Get "https://localhost:4443/": tls: failed to verify certificate: x509: certificate signed by unknown authority` {
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

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/02_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate), // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),  // PEM
	})
	helper.Must(err)
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}

	if !strings.HasPrefix(res.Header.Get("Location"), "https://") {
		t.Errorf("Expected https scheme in url variable value, got: %s", res.Header.Get("Location"))
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

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/03_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
	})
	helper.Must(err)
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

	_, err = client.Do(outreq)

	var RemoteFailedCertCheck = func(err error) bool {
		prefix := `Get "https://localhost:4443/": `
		suffix := "remote error: tls: bad certificate"
		return err != nil && (err.Error() == prefix+"readLoopPeekFailLocked: "+suffix ||
			err.Error() == prefix+suffix)
	}

	if !RemoteFailedCertCheck(err) {
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

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/04_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.ClientCertificate.Certificate),             // PEM
	})
	helper.Must(err)
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

	shutdown, hook, err := newCouperWithTemplate("testdata/mtls/04_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.CACertificate.Certificate),                 // PEM / just a non-matching one
	})
	helper.Must(err)
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

func TestHTTPSServer_TLS_ServerClientLeafOnly(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/06_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate), // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),  // PEM
		"clientLeaf": string(selfSigned.ClientCertificate.Certificate), // PEM
	})
	helper.Must(err)
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}
}

func TestHTTPSServer_TLS_ServerClientCertificateLeafMixed(t *testing.T) {
	helper := test.New(t)

	selfSigned1, err := server.NewCertificate(time.Hour, nil, nil)
	helper.Must(err)

	selfSigned2, err := server.NewCertificate(time.Hour, nil, nil)
	helper.Must(err)

	selfSigned3, err := server.NewCertificate(time.Hour, nil, nil)
	helper.Must(err)

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned1.CA.Leaf)

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/07_couper.hcl", helper, map[string]interface{}{
		"publicKey":    string(selfSigned1.ServerCertificate.Certificate),             // PEM
		"privateKey":   string(selfSigned1.ServerCertificate.PrivateKey),              // PEM
		"client1_Leaf": string(selfSigned1.ClientCertificate.Certificate),             // PEM
		"client2_CA":   string(selfSigned2.ClientIntermediateCertificate.Certificate), // PEM
		"client2_Leaf": string(selfSigned2.ClientCertificate.Certificate),             // PEM
		"client3_CA":   string(selfSigned3.ClientIntermediateCertificate.Certificate), // PEM
	})
	helper.Must(err)
	defer shutdown()

	for i, clientCert := range []*tls.Certificate{selfSigned1.Client, selfSigned2.Client, selfSigned3.Client} {
		t.Run(fmt.Sprintf("client%d", i+1), func(st *testing.T) {
			h := test.New(st)
			c := clientCert
			client := test.NewHTTPSClient(&tls.Config{
				RootCAs: pool,
				GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
					return c, nil
				},
			})

			outreq, e := http.NewRequest(http.MethodGet, "https://localhost:4443/", nil)
			h.Must(e)

			res, e := client.Do(outreq)
			h.Must(e)

			if res.StatusCode != http.StatusOK {
				st.Errorf("Expected statusOK, got: %d", res.StatusCode)
			}
		})
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

	shutdown, _, err := newCouperWithTemplate("testdata/mtls/05_couper.hcl", helper, map[string]interface{}{
		"publicKey":  string(selfSigned.ServerCertificate.Certificate),             // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),              // PEM
		"clientCA":   string(selfSigned.ClientIntermediateCertificate.Certificate), // PEM
		"clientLeaf": string(selfSigned.ClientCertificate.Certificate),             // PEM
		"clientKey":  string(selfSigned.ClientCertificate.PrivateKey),              // PEM
		"rootCA":     string(selfSigned.CACertificate.Certificate),                 // PEM
	})
	helper.Must(err)
	defer shutdown()

	outreq, err := http.NewRequest(http.MethodGet, "https://localhost:4443/inception", nil)
	helper.Must(err)

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected statusOK, got: %d", res.StatusCode)
	}
}
