package server

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/errors"
)

func defaultTLSConfig() *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		// We have to set this since otherwise Go will attempt to verify DNS names
		// match DNS SAN/CN which we don't want. We hook up VerifyPeerCertificate to
		// do our own path validation as well as Connect AuthZ.
		InsecureSkipVerify: true,
		// Include h2 to allow connect http servers to automatically support http2.
		// See: https://github.com/golang/go/blob/917c33fe8672116b04848cf11545296789cafd3b/src/net/http/server.go#L2724-L2731
		NextProtos: []string{"h2"},
	}
	return cfg
}

func requireClientAuth(config *config.ServerTLS) tls.ClientAuthType {
	if config != nil && len(config.ClientCertificate) > 0 {
		return tls.RequireAndVerifyClientCert
	}
	return tls.NoClientCert
}

func newTLSConfig(config *config.ServerTLS, log logrus.FieldLogger) (*tls.Config, error) {
	cfg := defaultTLSConfig()
	cfg.RootCAs = x509.NewCertPool() // no system CA's

	cfg.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		_, err := verifyChain(cfg, rawCerts)
		return err
	}

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
		cert, err := loadServerCertificate(certConfig)
		if err != nil {
			return nil, err
		}

		cfg.Certificates = append(cfg.Certificates, cert)
	}

	if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
		cfg.ClientCAs = x509.NewCertPool()
		for _, certConfig := range config.ClientCertificate {
			cert, _, err := loadClientCertificate(certConfig)
			if err != nil {
				return nil, err
			}
			cfg.ClientCAs.AddCert(cert.Leaf)
		}
	}

	// fallback to default built in cert
	if len(cfg.Certificates) == 0 {
		selfSigned, _ := NewCertificate(time.Hour*12, nil, nil)
		cfg.Certificates = append(cfg.Certificates, *selfSigned.Server)
	}

	return cfg, nil
}

func loadServerCertificate(config *config.ServerCertificate) (tls.Certificate, error) {
	if config.PrivateKeyFile != "" || config.PublicKeyFile != "" {
		wd, err := os.Getwd()
		if err != nil {
			return tls.Certificate{}, err
		}

		return tls.LoadX509KeyPair(filepath.Join(wd, config.PublicKeyFile), filepath.Join(wd, config.PrivateKeyFile))
	} else if config.PrivateKey != "" || config.PublicKey != "" {
		// With PEM block signature?
		certificate, err := tls.X509KeyPair([]byte(config.PublicKey), []byte(config.PrivateKey))
		if err == nil {
			return certificate, nil
		} // else assume DER
		x509certificate, err := x509.ParseCertificate([]byte(config.PublicKey))
		if err != nil {
			return tls.Certificate{}, err
		}
		key, err := x509.ParsePKCS8PrivateKey([]byte(config.PrivateKey))
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
	return tls.Certificate{}, errors.Configuration.Messagef("certificate %q: TODO: msg", config.Name)
}

func loadClientCertificate(config *config.ClientCertificate) (caCert tls.Certificate, leafCert tls.Certificate, err error) {
	var x509certificate *x509.Certificate

	wd, err := os.Getwd()
	if err != nil {
		return tls.Certificate{}, tls.Certificate{}, err
	}

	if config.CA != "" {
		x509certificate, err = x509.ParseCertificate([]byte(config.CA))
		if err != nil {
			return
		}

		caCert = tls.Certificate{
			Certificate: [][]byte{x509certificate.Raw},
			Leaf:        x509certificate,
		}
	} else if config.CAFile != "" {
		caCert, err = tls.LoadX509KeyPair(filepath.Join(wd, config.CAFile), "")
		if err != nil {
			return
		}
	}

	if config.Leaf != "" {
		x509certificate, err = x509.ParseCertificate([]byte(config.Leaf))
		if err != nil {
			return
		}

		leafCert = tls.Certificate{
			Certificate: [][]byte{x509certificate.Raw},
			Leaf:        x509certificate,
		}
	} else if config.LeafFile != "" {
		leafCert, err = tls.LoadX509KeyPair(filepath.Join(wd, config.LeafFile), "")
		if err != nil {
			return
		}
	}

	return
}

// verifyChain performs standard TLS verification without enforcing remote hostname matching.
func verifyChain(tlsCfg *tls.Config, rawCerts [][]byte) (*x509.Certificate, error) {
	// Fetch leaf and intermediates. This is based on code form tls handshake.
	if len(rawCerts) < 1 {
		return nil, errors.New().Message("tls: no certificates from peer")
	}
	certs := make([]*x509.Certificate, len(rawCerts))
	for i, asn1Data := range rawCerts {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return nil, errors.New().Message("tls: failed to parse certificate from peer: " + err.Error())
		}
		certs[i] = cert
	}

	cas := tlsCfg.RootCAs

	opts := x509.VerifyOptions{
		Roots:         cas,
		Intermediates: x509.NewCertPool(),
	}

	// Server side only sets KeyUsages in tls. This defaults to ServerAuth in
	// x509 lib. See
	// https://github.com/golang/go/blob/ee7dd810f9ca4e63ecfc1d3044869591783b8b74/src/crypto/x509/verify.go#L866-L868
	if tlsCfg.ClientCAs != nil {
		opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// All but the first cert are intermediates
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}
	_, err := certs[0].Verify(opts)
	return certs[0], err
}
