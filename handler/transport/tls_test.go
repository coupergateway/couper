package transport_test

import (
	"crypto/tls"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/handler/transport"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server"
)

func TestReadCertificates(t *testing.T) {
	helper := test.New(t)

	now := time.Now()
	selfSignedCert, serr := server.NewCertificate(time.Hour, nil, &now)
	helper.Must(serr)
	t.Logf("generated certificates in %s", time.Since(now).String())

	// obsolete for this test
	selfSignedCert.CA.PrivateKey = nil
	clientCertOnly := *selfSignedCert.Client
	clientCertOnly.PrivateKey = nil

	for _, format := range []string{"DER", "PEM"} {
		var caCertBytes, certBytes []byte
		switch format {
		case "DER":
			caCertBytes = selfSignedCert.CA.Certificate[0]
			certBytes = selfSignedCert.Client.Certificate[0]
		case "PEM":
			caCertBytes = selfSignedCert.CACertificate.Certificate
			certBytes = selfSignedCert.ClientCertificate.Certificate
		}

		pattern := "couper_test_tls_read_certs_" + format
		tmpCaCertFile, ferr := os.CreateTemp("", pattern)
		helper.Must(ferr)
		_, ferr = tmpCaCertFile.Write(caCertBytes)
		helper.Must(ferr)
		helper.Must(tmpCaCertFile.Close())
		defer os.Remove(tmpCaCertFile.Name())

		tmpCertFile, ferr := os.CreateTemp("", pattern)
		helper.Must(ferr)
		_, ferr = tmpCertFile.Write(certBytes)
		helper.Must(ferr)
		helper.Must(tmpCertFile.Close())
		defer os.Remove(tmpCertFile.Name())

		tests := []struct {
			name       string
			conf       config.BackendTLS
			wantSrv    tls.Certificate
			wantClient tls.Certificate
			wantErr    bool
		}{
			{"empty attributes", config.BackendTLS{}, tls.Certificate{}, tls.Certificate{}, false},
			{"server ca file", config.BackendTLS{ServerCertificateFile: tmpCaCertFile.Name()}, *selfSignedCert.CA, tls.Certificate{}, false},
			{"server ca value", config.BackendTLS{ServerCertificate: string(caCertBytes)}, *selfSignedCert.CA, tls.Certificate{}, false},
			{"server ca file + value", config.BackendTLS{ServerCertificateFile: tmpCaCertFile.Name(), ServerCertificate: string(caCertBytes)}, tls.Certificate{}, tls.Certificate{}, true},
			// TODO: testCase with combined crt+key PEM file
			{"client ca file /w malformed key", config.BackendTLS{ClientCertificateFile: tmpCertFile.Name(), ClientPrivateKey: "malformed"}, tls.Certificate{}, clientCertOnly, true},
			{"client ca file /w key", config.BackendTLS{ClientCertificateFile: tmpCertFile.Name(), ClientPrivateKey: string(selfSignedCert.ClientPrivateKey)}, tls.Certificate{}, *selfSignedCert.Client, false},
			{"client ca value /w key", config.BackendTLS{ClientCertificate: string(certBytes), ClientPrivateKey: string(selfSignedCert.ClientPrivateKey)}, tls.Certificate{}, *selfSignedCert.Client, false},
			{"client ca file /wo key", config.BackendTLS{ClientCertificateFile: tmpCertFile.Name()}, tls.Certificate{}, tls.Certificate{}, true},
			{"client ca value /wo key", config.BackendTLS{ClientCertificate: string(certBytes)}, tls.Certificate{}, tls.Certificate{}, true},
			{"client ca file + value", config.BackendTLS{ClientCertificateFile: tmpCertFile.Name(), ClientCertificate: string(certBytes)}, tls.Certificate{}, tls.Certificate{}, true},
		}
		for _, tt := range tests {
			t.Run(format+"/"+tt.name, func(t *testing.T) {
				gotSrv, gotClient, err := transport.ReadCertificates(&tt.conf)
				if (err != nil) != tt.wantErr {
					msg := "<nil>"
					if lerr, ok := err.(errors.GoError); ok {
						msg = lerr.LogError()
					} else if err != nil {
						msg = err.Error()
					}
					t.Errorf("ReadCertificates() error = %v, wantErr %v", msg, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(gotSrv, tt.wantSrv) {
					t.Errorf("ReadCertificates():\n\tgotSrv  %v\n\t\twant %v", gotSrv, tt.wantSrv)
				}
				if !reflect.DeepEqual(gotClient, tt.wantClient) {
					t.Errorf("ReadCertificates():\n\tgotClient %v\n\t\t want %v", gotClient, tt.wantClient)
				}
			})
		}
	}
}
