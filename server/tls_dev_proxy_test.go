package server_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

func TestNewTLSProxy(t *testing.T) {
	couperFile := []byte(`server "tls-dev" {
  endpoint "/" {
    response {
      status = 204
    }
  }
}

settings {
  https_dev_proxy = ["8443:8080", "9443:8080"]
}`)

	helper := test.New(t)
	shutdown, hook, err := newCouperWithBytes(couperFile, helper)
	helper.Must(err)
	defer func() {
		if t.Failed() {
			for _, e := range hook.AllEntries() {
				t.Log(e.String())
			}
		}
		shutdown()
	}()

	client := newClient()
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			cert, err := x509.ParseCertificate(rawCerts[0])
			helper.Must(err)
			if len(cert.DNSNames) == 0 || (cert.DNSNames[0] != "localhost" && cert.DNSNames[0] != "couper.dev") {
				t.Errorf("Expect dns name to be one of localhost,couper.dev; got: %v", cert.DNSNames)
			}
			return nil
		},
	}

	time.Sleep(2 * time.Second) // tls server needs some time.

	for _, p := range []string{"8443", "9443"} {
		for _, host := range []string{"127.0.0.1", "localhost", "couper.dev"} {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s:%s/", host, p), nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != http.StatusNoContent {
				t.Errorf("Expected 204, got: %d - %s", res.StatusCode, res.Status)
			}
		}
	}
}
