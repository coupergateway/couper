package server

import (
	"crypto/tls"
	"crypto/x509"
	"reflect"
	"testing"

	"github.com/avenga/couper/config"
)

func Test_requireClientAuth(t *testing.T) {
	tests := []struct {
		name   string
		config *config.ServerTLS
		want   tls.ClientAuthType
	}{
		{"NoClientCert without ClientCertificates", &config.ServerTLS{}, tls.NoClientCert},
		{"RequireAndVerifyClientCert with ClientCertificates", &config.ServerTLS{ClientCertificate: make([]*config.ClientCertificate, 1)}, tls.RequireAndVerifyClientCert},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := requireClientAuth(tt.config); got != tt.want {
				t.Errorf("requireClientAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_verifyChain(t *testing.T) {
	type args struct {
		tlsCfg   *tls.Config
		rawCerts [][]byte
	}
	tests := []struct {
		name    string
		args    args
		want    *x509.Certificate
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := verifyChain(tt.args.tlsCfg, tt.args.rawCerts)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyChain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("verifyChain() got = %v, want %v", got, tt.want)
			}
		})
	}
}
