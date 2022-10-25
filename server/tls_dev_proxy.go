package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler/middleware"
	"github.com/avenga/couper/logging"
	"github.com/avenga/couper/server/writer"
)

type ListenPort string
type Ports []string
type TLSDevPorts map[ListenPort]Ports

const TLSProxyOption = "https_dev_proxy"

var httpsDevProxyIDField = "x-" + xid.New().String()

func (tdp TLSDevPorts) Add(pair string) error {
	ports := strings.Split(pair, ":")
	if len(ports) != 2 {
		return errors.Configuration.Messagef("%s: invalid port mapping: %s", TLSProxyOption, pair)
	}
	for _, p := range ports {
		if _, err := strconv.Atoi(p); err != nil {
			return errors.Configuration.Messagef("%s: invalid format: %s", TLSProxyOption, pair).With(err)
		}
	}

	if dp, exist := tdp[ListenPort(ports[1])]; exist && dp.Contains(ports[0]) {
		return errors.Configuration.Messagef("https_dev_proxy: tls port already defined: %s", pair)
	}

	tdp[ListenPort(ports[1])] = append(tdp[ListenPort(ports[1])], ports[0])
	return nil
}

func (tdp TLSDevPorts) Get(p string) []string {
	if result, exist := tdp[ListenPort(p)]; exist {
		return result
	}
	return nil
}

func (tp Ports) Contains(needle string) bool {
	for _, p := range tp {
		if p == needle {
			return true
		}
	}
	return false
}

func (lp ListenPort) Port() runtime.Port {
	i, _ := strconv.Atoi(string(lp))
	return runtime.Port(i)
}

func NewTLSProxy(addr, port string, logger logrus.FieldLogger, settings *config.Settings) (*http.Server, error) {
	origin, err := url.Parse(fmt.Sprintf("http://%s/", addr))
	if err != nil {
		return nil, err
	}

	logEntry := logger.WithField("type", "couper_access_tls")

	httpProxy := httputil.NewSingleHostReverseProxy(origin)
	httpProxy.Transport = &http.Transport{ // http.DefaultTransport /wo Proxy
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	headers := []string{"Connection", "Upgrade"}
	accessLog := logging.NewAccessLog(&logging.Config{
		RequestHeaders:  append(logging.DefaultConfig.RequestHeaders, headers...),
		ResponseHeaders: append(logging.DefaultConfig.ResponseHeaders, headers...),
	}, logEntry)

	initialConfig, err := getTLSConfig(&tls.ClientHelloInfo{})
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp4", ":"+port)
	if err != nil {
		return nil, err
	}

	uidFn := middleware.NewUIDFunc(settings.RequestIDFormat)

	tlsServer := &http.Server{
		Addr:     ":" + port,
		ErrorLog: newErrorLogWrapper(logEntry),
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			uid := uidFn()
			req.Header.Set(httpsDevProxyIDField, uid)

			ctx := context.WithValue(req.Context(), request.ServerName, "couper_tls")
			ctx = context.WithValue(ctx, request.UID, uid)
			ctx = context.WithValue(ctx, request.StartTime, time.Now())

			req.Header.Set("Forwarded", fmt.Sprintf("for=%s;proto=https;host=%s;by=%s", req.RemoteAddr, req.Host, listener.Addr().String()))
			req.Header.Set("Via", "couper-https-dev-proxy")
			req.Header.Set("X-Forwarded-For", req.RemoteAddr+", "+listener.Addr().String())
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", "https")

			req.URL.Host = req.Host

			respW := writer.NewResponseWriter(rw, "")
			outreq := req.WithContext(ctx)
			httpProxy.ServeHTTP(respW, outreq)
			accessLog.Do(respW, outreq)
		}),
		TLSConfig: initialConfig,
	}

	go tlsServer.ServeTLS(listener, "", "")
	return tlsServer, err
}

var tlsConfigurations = map[string]*tls.Config{}
var tlsLock = sync.RWMutex{}

// getTLSConfig returns a clone from created or memorized tls configuration due to
// transport protocol upgrades / clones later on which would result in data races.
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

	tlsConfig, ok := tlsConfigurations[key]
	if !ok || tlsConfig.Certificates[0].Leaf.NotAfter.Before(time.Now()) {
		selfSigned, err := NewCertificate(time.Hour*24, hosts, nil)
		if err != nil {
			return nil, err
		}
		tlsConf := &tls.Config{
			Certificates:       []tls.Certificate{*selfSigned.Server},
			GetConfigForClient: getTLSConfig,
		}

		tlsConfigurations[key] = tlsConf
		return tlsConf.Clone(), nil
	}

	return tlsConfig.Clone(), nil
}

// ErrorWrapper logs incoming Write bytes with the context filled logrus.FieldLogger.
type ErrorWrapper struct{ l logrus.FieldLogger }

func (e *ErrorWrapper) Write(p []byte) (n int, err error) {
	msg := string(p)
	if strings.HasSuffix(msg, " tls: unknown certificate") ||
		strings.HasPrefix(msg, "http: TLS handshake error") {
		return len(p), nil // triggered on first browser connect for self signed certs; skip
	}
	e.l.Error(msg)
	return len(p), nil
}
func newErrorLogWrapper(logger logrus.FieldLogger) *log.Logger {
	return log.New(&ErrorWrapper{logger}, "", log.Lmsgprefix)
}
