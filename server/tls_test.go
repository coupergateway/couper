package server_test

import (
	"crypto/tls"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server"
)

func Test_LoadClientCertificate(t *testing.T) {
	helper := test.New(t)

	now := time.Now()
	selfSignedCert, serr := server.NewCertificate(time.Hour, nil, &now)
	helper.Must(serr)
	t.Logf("generated certificates in %s", time.Since(now).String())

	// obsolete for this test
	selfSignedCert.ClientIntermediate.PrivateKey = nil
	selfSignedCert.Client.PrivateKey = nil

	for _, format := range []string{"DER", "PEM"} {
		var caCertBytes, certBytes []byte
		switch format {
		case "DER":
			caCertBytes = selfSignedCert.ClientIntermediate.Certificate[0]
			certBytes = selfSignedCert.Client.Certificate[0]
		case "PEM":
			caCertBytes = selfSignedCert.ClientIntermediateCertificate.Certificate
			certBytes = selfSignedCert.ClientCertificate.Certificate
		}

		pattern := "couper_test_tls_" + format
		tmpCertFile, err := os.CreateTemp("", pattern)
		helper.Must(err)
		_, err = tmpCertFile.Write(caCertBytes)
		helper.Must(err)
		helper.Must(tmpCertFile.Close())
		defer os.Remove(tmpCertFile.Name())

		tmpLeafCertFile, err := os.CreateTemp("", pattern)
		helper.Must(err)
		_, err = tmpLeafCertFile.Write(certBytes)
		helper.Must(err)
		helper.Must(tmpLeafCertFile.Close())
		defer os.Remove(tmpLeafCertFile.Name())

		tests := []struct {
			name         string
			config       *config.ClientCertificate
			wantCaCert   tls.Certificate
			wantLeafCert tls.Certificate
			wantErr      bool
		}{
			{"nil clientCertificate", nil, tls.Certificate{}, tls.Certificate{}, false},
			{"empty clientCertificate", &config.ClientCertificate{}, tls.Certificate{}, tls.Certificate{}, true},
			{"malformed clientCertificate value", &config.ClientCertificate{CA: "asdf"}, tls.Certificate{}, tls.Certificate{}, true},
			{"clientCertificate CA value", &config.ClientCertificate{
				CA: string(caCertBytes),
			}, *selfSignedCert.ClientIntermediate, tls.Certificate{}, false},
			{"clientCertificate CA /w Leaf value", &config.ClientCertificate{
				CA:   string(caCertBytes),
				Leaf: string(certBytes),
			}, *selfSignedCert.ClientIntermediate, *selfSignedCert.Client, false},
			{"clientCertificate /w Leaf value", &config.ClientCertificate{
				Leaf: string(certBytes),
			}, tls.Certificate{}, *selfSignedCert.Client, false},
		}
		for _, tt := range tests {
			t.Run(format+"/"+tt.name, func(t *testing.T) {
				gotCaCert, gotLeafCert, err := server.LoadClientCertificate(tt.config)
				if (err != nil) != tt.wantErr {
					msg := err.Error()
					if cerr, ok := err.(errors.GoError); ok {
						msg = cerr.LogError()
					}
					t.Errorf("LoadClientCertificate() error = %v, wantErr %v", msg, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(gotCaCert, tt.wantCaCert) {
					t.Errorf("LoadClientCertificate() CA\n\tgot:\t%v\n\twant:\t%v\n", gotCaCert, tt.wantCaCert)
				}
				if !reflect.DeepEqual(gotLeafCert, tt.wantLeafCert) {
					t.Errorf("LoadClientCertificate() Leaf\n\tgot:\t%v\n\twant:\t%v\n", gotLeafCert, tt.wantLeafCert)
				}
			})
		}
	}
}

func Test_LoadServerCertificate(t *testing.T) {
	helper := test.New(t)

	now := time.Now()
	selfSignedCert, serr := server.NewCertificate(time.Hour, nil, &now)
	helper.Must(serr)
	t.Logf("generated certificates in %s", time.Since(now).String())

	for _, format := range []string{"DER", "PEM"} {
		var certBytes, privateKeyBytes []byte
		switch format {
		case "DER":
			certBytes = selfSignedCert.Server.Certificate[0]
			privateKeyBytes = selfSignedCert.ServerPrivateKey
		case "PEM":
			certBytes = selfSignedCert.ServerCertificate.Certificate
			privateKeyBytes = selfSignedCert.ServerCertificate.PrivateKey
		}

		pattern := "couper_test_tls_" + format
		tmpCertFile, err := os.CreateTemp("", pattern)
		helper.Must(err)
		_, err = tmpCertFile.Write(certBytes)
		helper.Must(err)
		helper.Must(tmpCertFile.Close())
		defer os.Remove(tmpCertFile.Name())

		tmpKeyFile, err := os.CreateTemp("", pattern)
		helper.Must(err)
		_, err = tmpKeyFile.Write(privateKeyBytes)
		helper.Must(err)
		helper.Must(tmpKeyFile.Close())
		defer os.Remove(tmpKeyFile.Name())

		tests := []struct {
			name    string
			config  *config.ServerCertificate
			want    tls.Certificate
			wantErr bool
		}{
			{"nil serverCertificate", nil, tls.Certificate{}, false},
			{"empty serverCertificate", &config.ServerCertificate{}, tls.Certificate{}, true},
			{"with serverCertificateValue", &config.ServerCertificate{
				PublicKey:  string(certBytes),
				PrivateKey: string(privateKeyBytes),
			}, *selfSignedCert.Server, false},
			{"with serverCertificateFile", &config.ServerCertificate{
				PublicKeyFile:  tmpCertFile.Name(),
				PrivateKeyFile: tmpKeyFile.Name(),
			}, *selfSignedCert.Server, false},
		}
		for _, tt := range tests {
			t.Run(format+"/"+tt.name, func(t *testing.T) {
				got, terr := server.LoadServerCertificate(tt.config)
				if (terr != nil) != tt.wantErr {
					t.Errorf("LoadServerCertificate() error = %v, wantErr %v", terr, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("LoadServerCertificate()\n\tgot:\t%v\n\twant:\t%v\n", got, tt.want)
				}
			})
		}
	}
}
