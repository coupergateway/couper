package server

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
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
	var leafCert []byte

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
				leafCert = clientCrt.Leaf.Raw
			}
			// TODO: verify clientCrt with cert ?
		}
	}

	if len(leafCert) > 0 {
		cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if !bytes.Equal(rawCerts[0], leafCert) {
				return fmt.Errorf("tls: client leaf certificate mismatch")
			}
			return nil
		}
	}

	// fallback to default built in cert
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
	cert, err := coupertls.ReadValueOrFile(config.PublicKey, config.PublicKeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	privateKey, err := coupertls.ReadValueOrFile(config.PrivateKey, config.PrivateKeyFile)
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

	caCert, err := coupertls.ReadValueOrFile(config.CA, config.CAFile)
	if err != nil && !hasLeaf {
		return fail(err)
	}

	leafCert, err := coupertls.ReadValueOrFile(config.Leaf, config.LeafFile)
	if err != nil && hasLeaf { // since its optional, if given
		return fail(err)
	}

	// try PEM encoded
	var leafCertificate tls.Certificate
	if len(leafCert) > 0 {
		leafCertificate, err = coupertls.ParseCertificate(leafCert, nil)
	}

	if len(caCert) == 0 && leafCertificate.Leaf != nil {
		return tls.Certificate{}, leafCertificate, nil
	}

	caCertificate, err := coupertls.ParseCertificate(caCert, nil)

	return caCertificate, leafCertificate, err
}
