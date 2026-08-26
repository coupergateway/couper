package authzen

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"time"
)

// metadataTLS holds the TLS connection state of the client request. Couper sends it as a
// fact about the request, not as a statement about the principal: the certificate can
// belong to a mesh sidecar while a bearer token identifies the caller.
type metadataTLS struct {
	CipherSuite       string             `json:"cipher_suite"`
	ClientCertificate *clientCertificate `json:"client_certificate,omitempty"`
	ServerName        string             `json:"server_name,omitempty"`
	Version           string             `json:"version"`
}

// clientCertificate carries the fields an authorization service keys on for a client-facing
// mTLS decision: the subject/issuer DN, the serial and SHA-256 fingerprint for allow lists or
// pinning, validity, and the subject alternative names that often hold the real identity.
type clientCertificate struct {
	DNSNames          []string  `json:"dns_names,omitempty"`
	EmailAddresses    []string  `json:"email_addresses,omitempty"`
	FingerprintSHA256 string    `json:"fingerprint_sha256,omitempty"`
	IPAddresses       []string  `json:"ip_addresses,omitempty"`
	Issuer            string    `json:"issuer"`
	NotAfter          time.Time `json:"not_after"`
	NotBefore         time.Time `json:"not_before"`
	SerialNumber      string    `json:"serial_number,omitempty"`
	Subject           string    `json:"subject"`
	URIs              []string  `json:"uris,omitempty"`
}

func newMetadataTLS(state *tls.ConnectionState) *metadataTLS {
	if state == nil {
		return nil
	}

	meta := &metadataTLS{
		CipherSuite: tls.CipherSuiteName(state.CipherSuite),
		ServerName:  state.ServerName,
		Version:     tls.VersionName(state.Version),
	}

	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		clientCert := &clientCertificate{
			DNSNames:       cert.DNSNames,
			EmailAddresses: cert.EmailAddresses,
			Issuer:         cert.Issuer.String(),
			NotAfter:       cert.NotAfter,
			NotBefore:      cert.NotBefore,
			Subject:        cert.Subject.String(),
		}
		if cert.SerialNumber != nil {
			clientCert.SerialNumber = cert.SerialNumber.Text(16)
		}
		if len(cert.Raw) > 0 {
			sum := sha256.Sum256(cert.Raw)
			clientCert.FingerprintSHA256 = hex.EncodeToString(sum[:])
		}
		for _, uri := range cert.URIs {
			clientCert.URIs = append(clientCert.URIs, uri.String())
		}
		for _, ip := range cert.IPAddresses {
			clientCert.IPAddresses = append(clientCert.IPAddresses, ip.String())
		}
		meta.ClientCertificate = clientCert
	}

	return meta
}
