package server

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	coupertls "github.com/avenga/couper/internal/tls"
)

func requireClientAuth(config *config.ServerTLS) tls.ClientAuthType {
	if config != nil && len(config.ClientCertificate) > 0 {
		return tls.RequireAndVerifyClientCert
	}
	return tls.NoClientCert
}

func newTLSConfig(config *config.ServerTLS, log logrus.FieldLogger) (*tls.Config, error) {
	cfg := coupertls.DefaultTLSConfig()
	cfg.RootCAs = x509.NewCertPool() // no system CA's
	var leafOnlyCerts [][]byte

	cfg.GetCertificate = func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		log.WithField("ClientHelloInfo", logrus.Fields{
			"connection": logrus.Fields{
				"client_ip": info.Conn.RemoteAddr().String(),
				"server_ip": info.Conn.LocalAddr().String(),
			},
			"server_name":      info.ServerName,
			"supported_protos": info.SupportedProtos,
		}).Debug()
		return nil, nil
	}

	cfg.ClientAuth = requireClientAuth(config)

	for _, certConfig := range config.ServerCertificates {
		cert, err := LoadServerCertificate(certConfig)
		if err != nil {
			return nil, err
		}

		cfg.Certificates = append(cfg.Certificates, cert)
	}

	var leafCerts map[string][]byte
	if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
		cfg.ClientCAs = x509.NewCertPool()
		for _, certConfig := range config.ClientCertificate {
			cert, clientCrt, err := LoadClientCertificate(certConfig)
			if err != nil {
				return nil, err
			}

			if cert.Leaf != nil {
				cfg.ClientCAs.AddCert(cert.Leaf)
			}

			if clientCrt.Leaf != nil {
				if cert.Leaf == nil {
					leafOnlyCerts = append(leafOnlyCerts, clientCrt.Leaf.Raw)
				} else {
					if leafCerts == nil {
						leafCerts = make(map[string][]byte)
					}
					leafCerts[checksum(cert.Leaf.Raw)] = clientCrt.Leaf.Raw
				}
			}
		}
	}

	if len(leafCerts)+len(leafOnlyCerts) > 0 {
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// TODO: check if chains can be passed to prevent double verify
			return verifyClientCertificate(cfg.ClientCAs, leafCerts, leafOnlyCerts, rawCerts)
		}

		// If one of the certificate blocks has no CA certificate but a leaf one then downgrade
		// the clientCert option to prevent the tls module to call verify on its own.
		if len(leafOnlyCerts) > 0 {
			cfg.ClientAuth = tls.RequestClientCert
		}
	}

	// fallback to the self-signed server certificate
	if len(cfg.Certificates) == 0 {
		selfSigned, _ := NewCertificate(time.Hour*12, nil, nil)
		cfg.Certificates = append(cfg.Certificates, *selfSigned.Server)
	}

	return cfg, nil
}

func LoadServerCertificate(config *config.ServerCertificate) (tls.Certificate, error) {
	if config == nil {
		return tls.Certificate{}, nil // currently triggers self signed fallback
	}
	cert, err := reader.ReadFromAttrFile("tls", config.PublicKey, config.PublicKeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	privateKey, err := reader.ReadFromAttrFile("tls", config.PrivateKey, config.PrivateKeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Try PEM encoded
	certificate, err := tls.X509KeyPair(cert, privateKey)
	if err == nil {
		certificate.Leaf, err = x509.ParseCertificate(certificate.Certificate[0])
		return certificate, err
	} // otherwise assume DER
	x509certificate, err := x509.ParseCertificate(cert)
	if err != nil {
		return tls.Certificate{}, err
	}
	key, err := x509.ParsePKCS8PrivateKey(privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// TODO: Check for public <> private key type match as seen in tls.X509KeyPair() method.
	return tls.Certificate{
		Certificate: [][]byte{x509certificate.Raw},
		Leaf:        x509certificate,
		PrivateKey:  key,
	}, nil
}

func LoadClientCertificate(config *config.ClientCertificate) (tls.Certificate, tls.Certificate, error) {
	fail := func(err error) (tls.Certificate, tls.Certificate, error) {
		return tls.Certificate{}, tls.Certificate{}, errors.Configuration.With(err).Message("client_certificate:")
	}

	if config == nil {
		return tls.Certificate{}, tls.Certificate{}, nil
	}

	hasLeaf := config.Leaf != "" || config.LeafFile != ""

	caCert, err := reader.ReadFromAttrFile("tls", config.CA, config.CAFile)
	if err != nil && !hasLeaf {
		return fail(err)
	}

	leafCert, err := reader.ReadFromAttrFile("tls", config.Leaf, config.LeafFile)
	if err != nil && hasLeaf { // since its optional, if given
		return fail(err)
	}

	// try PEM encoded
	var leafCertificate tls.Certificate
	if len(leafCert) > 0 {
		leafCertificate, err = coupertls.ParseCertificate(leafCert, nil)
		if err != nil {
			return fail(err)
		}
	}

	if len(caCert) == 0 && leafCertificate.Leaf != nil {
		return tls.Certificate{}, leafCertificate, nil
	}

	caCertificate, err := coupertls.ParseCertificate(caCert, nil)

	return caCertificate, leafCertificate, err
}

// verifyClientCertificate will be called as soon as a client_certificate block contains just one leaf certificate
// without a CA certificate. Other possible client_certificate blocks with CA certificate must still be validated.
// If there is no "leaf-only" client_certificate block the tls package will do the verification.
func verifyClientCertificate(caPool *x509.CertPool, leafs map[string][]byte, leafsOnly [][]byte, rawCerts [][]byte) error {
	for _, leaf := range leafsOnly {
		for _, cert := range rawCerts {
			if bytes.Equal(cert, leaf) {
				return nil
			}
		}
	}

	// unfortunately parsing happens twice, already before calling this callback
	certs := make([]*x509.Certificate, len(rawCerts))
	var err error
	for i, asn1Data := range rawCerts {
		if certs[i], err = x509.ParseCertificate(asn1Data); err != nil {
			return fmt.Errorf("tls: failed to parse client certificate: " + err.Error())
		}
	}

	chains, err := verify(caPool, certs)
	if err != nil {
		return err
	}

	// Determine the requirement to verify leaf equality with CA <> leaf mapping.
	var checkLeaf bool
	var leaf []byte
	if leafs != nil && len(chains) > 0 && len(chains[0]) > 1 {
		leaf, checkLeaf = leafs[checksum(chains[0][1].Raw)]
	}

	if !checkLeaf {
		return nil
	}

	for _, cert := range rawCerts {
		if bytes.Equal(cert, leaf) {
			return nil
		}
	}
	return fmt.Errorf("tls: client leaf certificate mismatch")
}

func verify(caPool *x509.CertPool, certs []*x509.Certificate) ([][]*x509.Certificate, error) {
	opts := x509.VerifyOptions{
		Roots:         caPool,
		CurrentTime:   time.Now(),
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}

	chains, err := certs[0].Verify(opts)
	if err != nil {
		return nil, fmt.Errorf("tls: failed to verify client certificate: " + err.Error())
	}
	return chains, nil
}

func checksum(b []byte) string {
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}
