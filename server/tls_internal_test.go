package server

import (
	"crypto/tls"
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
