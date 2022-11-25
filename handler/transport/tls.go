package transport

import (
	"crypto/tls"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/reader"
	"github.com/avenga/couper/errors"
	coupertls "github.com/avenga/couper/internal/tls"
)

// ReadCertificates parses an optional CA certificate or a client certificate / key pair.
// It is valid to have just the client pair without the CA certificate since the system
// Root CAs or the related Couper cli option MAY configure the related transport too.
func ReadCertificates(conf *config.BackendTLS) (tls.Certificate, tls.Certificate, error) {
	fail := func(err error) (tls.Certificate, tls.Certificate, error) {
		return tls.Certificate{}, tls.Certificate{}, err
	}

	if conf == nil {
		return fail(nil)
	}

	hasCA := conf.ServerCertificate != "" || conf.ServerCertificateFile != ""
	hasClient := conf.ClientCertificate != "" || conf.ClientCertificateFile != ""
	hasClientKey := conf.ClientPrivateKey != "" || conf.ClientPrivateKeyFile != ""

	if !hasCA && !hasClient {
		return fail(nil)
	}

	if hasClient && !hasClientKey {
		return fail(errors.Configuration.Message("tls: missing client private key"))
	}

	var caCertificate, clientCertificate tls.Certificate

	caCert, err := reader.ReadFromAttrFile("tls", conf.ServerCertificate, conf.ServerCertificateFile)
	if err != nil && hasCA {
		return fail(err)
	}

	clientCert, err := reader.ReadFromAttrFile("tls", conf.ClientCertificate, conf.ClientCertificateFile)
	if err != nil && hasClient {
		return fail(err)
	}

	clientKey, err := reader.ReadFromAttrFile("tls", conf.ClientPrivateKey, conf.ClientPrivateKeyFile)
	if err != nil && (conf.ClientPrivateKey != "" || conf.ClientPrivateKeyFile != "") {
		return fail(err)
	}

	if len(caCert) > 0 {
		caCertificate, err = coupertls.ParseCertificate(caCert, nil)
		if err != nil {
			return fail(err)
		}
	}

	if len(clientCert) > 0 {
		clientCertificate, err = coupertls.ParseCertificate(clientCert, clientKey)
	}

	return caCertificate, clientCertificate, err
}
