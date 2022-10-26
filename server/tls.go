package server

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
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
		cert, err := LoadServerCertificate(certConfig)
		if err != nil {
			return nil, err
		}

		cfg.Certificates = append(cfg.Certificates, cert)
	}

	if cfg.ClientAuth == tls.RequireAndVerifyClientCert {
		cfg.ClientCAs = x509.NewCertPool()
		for _, certConfig := range config.ClientCertificate {
			cert, _, err := LoadClientCertificate(certConfig)
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

func LoadServerCertificate(config *config.ServerCertificate) (tls.Certificate, error) {
	if config == nil {
		return tls.Certificate{}, nil // currently triggers self signed fallback
	}
	cert, err := readValueOrFile(config.PublicKey, config.PublicKeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}

	privateKey, err := readValueOrFile(config.PrivateKey, config.PrivateKeyFile)
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
		return tls.Certificate{}, tls.Certificate{}, err
	}
	if config == nil {
		return fail(nil)
	}

	var x509certificate *x509.Certificate

	caCert, err := readValueOrFile(config.CA, config.CAFile)
	if err != nil {
		return fail(err)
	}

	leafCert, err := readValueOrFile(config.Leaf, config.LeafFile)
	if err != nil && (config.Leaf != "" || config.LeafFile != "") { // since its optional, if given
		return fail(err)
	}

	// try PEM encoded
	var caCertificate tls.Certificate
	certDERBlock, _ := pem.Decode(caCert)
	if certDERBlock != nil {
		caCertificate.Certificate = append(caCertificate.Certificate, certDERBlock.Bytes)
		caCertificate.Leaf, err = x509.ParseCertificate(certDERBlock.Bytes)
		if err != nil {
			return fail(err)
		}
	} else { // assume DER
		x509certificate, err = x509.ParseCertificate(caCert)
		if err != nil {
			return fail(err)
		}

		caCertificate = tls.Certificate{
			Certificate: [][]byte{caCert},
			Leaf:        x509certificate,
		}
	}

	if len(leafCert) == 0 {
		return caCertificate, tls.Certificate{}, nil
	}

	// try PEM encoded
	var leafCertificate tls.Certificate
	certDERBlock, _ = pem.Decode(leafCert)
	if certDERBlock != nil {
		leafCertificate.Certificate = append(leafCertificate.Certificate, certDERBlock.Bytes)
		leafCertificate.Leaf, err = x509.ParseCertificate(certDERBlock.Bytes)
		if err != nil {
			return fail(err)
		}
	} else { // assume DER
		x509certificate, err = x509.ParseCertificate(leafCert)
		if err != nil {
			return fail(err)
		}

		leafCertificate = tls.Certificate{
			Certificate: [][]byte{leafCert},
			Leaf:        x509certificate,
		}
	}

	return caCertificate, leafCertificate, nil
}

func readValueOrFile(value, filePath string) ([]byte, error) {
	if value != "" && filePath != "" {
		return nil, errors.New().Message("both attributes provided")
	} else if value != "" {
		return []byte(value), nil
	} else if filePath != "" {
		return os.ReadFile(filePath)
	}
	return nil, errors.New().Message("no attributes provided")
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
