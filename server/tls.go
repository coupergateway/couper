package server

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
)

func NewTLSProxy(addr, port string, logger logrus.FieldLogger) (*http.Server, error) {
	origin, err := url.Parse(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		return nil, err
	}

	log := logger.WithField("type", "couper_access_tls")

	httpProxy := httputil.NewSingleHostReverseProxy(origin)

	headers := []string{"Connection", "Upgrade"}
	accessLog := logging.NewAccessLog(&logging.Config{RequestHeaders: headers, ResponseHeaders: headers}, log)

	initialConfig, err := getTLSConfig(&tls.ClientHelloInfo{})
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		return nil, err
	}

	tlsServer := &http.Server{
		Addr:     ":" + port,
		ErrorLog: newErrorLogWrapper(log),
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), request.ServerName, "couper_tls")
			ctx = context.WithValue(ctx, request.UID, xid.New())
			req.URL.Host = req.Host
			respW := writer.NewResponseWriter(rw, "")
			accessLog.ServeHTTP(respW, req.WithContext(ctx), httpProxy, time.Now())
		}),
		TLSConfig: initialConfig,
	}

	go tlsServer.ServeTLS(listener, "", "")
	return tlsServer, err
}

var tlsConfigurations = sync.Map{}
var tlsLock = sync.RWMutex{}

func getTLSConfig(info *tls.ClientHelloInfo) (*tls.Config, error) {
	var hosts []string
	key := "localhost"
	if info.ServerName != "" {
		hosts = append(hosts, info.ServerName)
		key = info.ServerName
	}

	// Global lock to prevent recreate loop for new connections.
	tlsLock.Lock()
	defer tlsLock.Unlock()

	storedCert, ok := tlsConfigurations.Load(key)
	if !ok {
		cert, _, err := newCertificate(time.Hour*24, hosts, nil)
		if err != nil {
			return nil, err
		}
		tlsConf := &tls.Config{
			Certificates:       []tls.Certificate{*cert},
			GetConfigForClient: getTLSConfig,
		}

		tlsConfigurations.Store(key, tlsConf)
		return tlsConf, nil
	}

	return storedCert.(*tls.Config), nil
}

// newCertificate creates a certificate with given host and duration.
// If no hosts are provided all localhost variants will be used.
func newCertificate(duration time.Duration, hosts []string, notBefore *time.Time) (*tls.Certificate, *x509.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	if len(hosts) == 0 {
		hosts = []string{"127.0.0.1", "::1", "localhost", "0.0.0.0", "::0"}
	}

	if notBefore == nil {
		n := time.Now()
		notBefore = &n
	}
	notAfter := notBefore.Add(duration)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:       []string{"Couper"},
			OrganizationalUnit: []string{"Development"},
		},
		NotBefore: *notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// self CA
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	certOut := &bytes.Buffer{}
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return nil, nil, err
	}

	keyOut := &bytes.Buffer{}
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	if err != nil {
		return nil, nil, err
	}
	cert, err := tls.X509KeyPair(certOut.Bytes(), keyOut.Bytes())
	if err != nil {
		return nil, nil, err
	}
	x509cert, err := x509.ParseCertificate(derBytes)
	return &cert, x509cert, err
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// ErrorWrapper logs incoming Write bytes with the context filled logrus.FieldLogger.
// Created to keep the ReverseProxy file as clean as possible and dependency free.
type ErrorWrapper struct{ l logrus.FieldLogger }

func (e *ErrorWrapper) Write(p []byte) (n int, err error) {
	msg := strings.Replace(string(p), "\n", "", 1)
	if strings.HasSuffix(msg, " tls: unknown certificate") {
		e.l.Warn(msg)
	} else {
		e.l.Error(msg)
	}
	return len(p), nil
}
func newErrorLogWrapper(logger logrus.FieldLogger) *log.Logger {
	return log.New(&ErrorWrapper{logger}, "", log.Lmsgprefix)
}
