package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"github.com/coupergateway/couper/errors"
)

// ParseCertificate reads a certificate from the given bytes.
// Either as PEM format where chained ones are considered or just plain DER format.
func ParseCertificate(cert, key []byte) (certificate tls.Certificate, err error) {
	certBytes := cert[:]
	// try PEM format
	var keyDERBlock *pem.Block
	for {
		var certDERBlock *pem.Block
		certDERBlock, certBytes = pem.Decode(certBytes)
		if certDERBlock == nil {
			if len(certificate.Certificate) > 0 {
				certificate.Leaf, err = x509.ParseCertificate(certificate.Certificate[0])
				if err != nil {
					return certificate, err
				}

				// Option to combine both into one file.
				if keyDERBlock != nil {
					certificate.PrivateKey, err = parsePrivateKey(keyDERBlock.Bytes)
					if err != nil {
						return certificate, err
					}
				}
			}
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			certificate.Certificate = append(certificate.Certificate, certDERBlock.Bytes)
		} else if strings.HasSuffix(certDERBlock.Type, " PRIVATE KEY") {
			keyDERBlock = certDERBlock
		}
	}

	// assume DER format
	if len(certificate.Certificate) == 0 {
		var x509Certificate *x509.Certificate
		x509Certificate, err = x509.ParseCertificate(cert)
		if err != nil {
			return certificate, err
		}

		certificate = tls.Certificate{
			Certificate: [][]byte{cert},
			Leaf:        x509Certificate,
		}
	}

	if certificate.PrivateKey == nil && len(key) > 0 {
		keyBytes := key[:]
		keyDERBlock, _ = pem.Decode(key)
		if keyDERBlock != nil {
			keyBytes = keyDERBlock.Bytes
		}
		certificate.PrivateKey, err = parsePrivateKey(keyBytes)
	}

	return certificate, err
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS #1 private keys by default, while OpenSSL 1.0.0 generates PKCS #8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key, nil
		default:
			return nil, errors.Configuration.Message("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.Configuration.Message("tls: failed to parse private key")
}
