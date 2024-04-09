package test

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/coupergateway/couper/server"
)

type Backend struct {
	srv *httptest.Server
	mux *http.ServeMux
}

func NewBackend() *Backend {
	b := &Backend{
		mux: http.NewServeMux(),
	}

	b.srv = httptest.NewServer(b)

	registerHTTPHandler(b)

	return b
}

func (b *Backend) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	b.mux.ServeHTTP(rw, req)
}

func (b *Backend) Close() {
	b.srv.Close()
}

func (b *Backend) Addr() string {
	return b.srv.URL
}

func NewExpiredBackend() (*Backend, *server.SelfSignedCertificate) {
	b := &Backend{
		mux: http.NewServeMux(),
	}

	b.srv = httptest.NewUnstartedServer(b.mux)

	selfSigned, err := server.NewCertificate(time.Microsecond, nil, nil)
	if err != nil {
		panic(err)
	}

	b.srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{*selfSigned.Server},
	}

	registerHTTPHandler(b)

	b.srv.StartTLS()
	return b, selfSigned
}

func registerHTTPHandler(b *Backend) {
	// test handler
	b.mux.HandleFunc("/", createAnythingHandler(http.StatusNotFound))
	b.mux.HandleFunc("/.well-known/openid-configuration", oidc)
	b.mux.HandleFunc("/anything", createAnythingHandler(http.StatusOK))
	b.mux.HandleFunc("/anything/", createAnythingHandler(http.StatusOK))
	b.mux.HandleFunc("/error", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	b.mux.HandleFunc("/health", health)
	b.mux.HandleFunc("/jwks.json", jwks)
	b.mux.HandleFunc("/pdf", pdf)
	b.mux.HandleFunc("/redirect", redirect)
	b.mux.HandleFunc("/reflect", reflect)
	b.mux.HandleFunc("/reflectDelay", reflectDelay)
	b.mux.HandleFunc("/small", small)
	b.mux.HandleFunc("/ws", echo)
}

func createAnythingHandler(status int) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		type anything struct {
			Args, Query, PostForm              url.Values
			Body                               string
			Headers                            http.Header
			Host                               string
			JSON                               map[string]interface{}
			Method, RemoteAddr, URL, UserAgent string
			Path                               string
			RawQuery                           string
			ResponseStatus                     int
		}

		_ = req.ParseForm()

		body, _ := io.ReadAll(req.Body)

		resp := &anything{
			Args:           req.Form,
			Body:           string(body),
			Headers:        req.Header.Clone(),
			Host:           req.Host,
			Method:         req.Method,
			Path:           req.URL.Path,
			PostForm:       req.PostForm,
			Query:          req.URL.Query(),
			RawQuery:       req.URL.RawQuery,
			RemoteAddr:     req.RemoteAddr,
			ResponseStatus: status,
			URL:            req.URL.String(),
			UserAgent:      req.UserAgent(),
			JSON: map[string]interface{}{
				"list": []string{"item1", "item2"},
			},
		}

		respContent, _ := json.Marshal(resp)

		rw.Header().Set("Server", "couper test-backend")
		rw.Header().Set("Date", time.Now().Format(http.TimeFormat))
		rw.Header().Set("Content-Length", strconv.Itoa(len(respContent)))
		rw.Header().Set("Content-Type", "application/json")

		rw.Header().Set("Remove-Me-1", "r1")
		rw.Header().Set("Remove-Me-2", "r2")

		if delay := req.URL.Query().Get("delay"); delay != "" {
			delayDuration, _ := time.ParseDuration(delay)
			if delayDuration > 0 {
				time.Sleep(delayDuration)
			}
		}

		rw.WriteHeader(status)
		_, _ = rw.Write(respContent)
	}
}

// echo handler expected a "ping" after sending the ws upgrade header and response with a "pong" once.
func echo(rw http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Connection") != "upgrade" &&
		req.Header.Get("Upgrade") != "websocket" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	hj, ok := rw.(http.Hijacker)
	if !ok {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	conn, _, err := hj.Hijack()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	defer conn.Close()

	// Clear deadlines
	conn.SetDeadline(time.Time{})

	if req.Header.Get("Echo") == "ECHO" {
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nEcho: ECHO\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	} else {
		_, _ = conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"))
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func(c net.Conn) {
		time.Sleep(time.Second)

		p := make([]byte, 4)
		conn.Read(p)
		if string(p) == "ping" {
			conn.Write([]byte("pong"))
			wg.Done()
		}
	}(conn)

	wg.Wait()
}

func redirect(rw http.ResponseWriter, req *http.Request) {
	queryParams := req.URL.Query()
	if queryParams.Get("url") == "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Allow relative redirects only
	uri, _ := url.ParseRequestURI(queryParams.Get("url"))
	uri.Scheme = ""
	uri.Host = ""

	rw.Header().Set("Location", uri.String())
	rw.WriteHeader(http.StatusFound)
}

func pdf(rw http.ResponseWriter, req *http.Request) {
	_, currFile, _, _ := runtime.Caller(0)
	rootDir := http.Dir(path.Join(path.Dir(currFile), "testdata"))

	file, err := rootDir.Open("blank.pdf")
	if err != nil {
		return
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return
	}

	http.ServeContent(rw, req, "/blank.pdf", info.ModTime(), file)
}

func small(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("1234567890"))
}

func health(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte(" 👍 "))
}

func jwks(rw http.ResponseWriter, req *http.Request) {
	destruct := req.Header.Get("Self-Destruct")
	if destruct != "" {
		seconds, _ := strconv.ParseInt(destruct, 10, 64)
		if time.Now().After(time.Unix(seconds, 0)) {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(`{"booom":1}`))
			return
		}
	}

	_, currFile, _, _ := runtime.Caller(0)
	rootDir := http.Dir(path.Join(path.Dir(currFile), "testdata"))

	file, err := rootDir.Open("jwks.json")
	if err != nil {
		return
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return
	}

	http.ServeContent(rw, req, "/jwks.json", info.ModTime(), file)
}

func oidc(rw http.ResponseWriter, req *http.Request) {
	body := []byte(`{
			"authorization_endpoint": "https://authorization.server/oauth2/authorize",
			"issuer": "https://authorization.server",
			"jwks_uri": "http://` + req.Host + `/jwks.json",
			"token_endpoint": "http://` + req.Host + `/token",
			"userinfo_endpoint": "http://` + req.Host + `/userinfo"
			}`)
	_, _ = rw.Write(body)
}

func reflect(rw http.ResponseWriter, req *http.Request) {
	for k, v := range req.Header {
		rw.Header()[k] = v
	}
	_, _ = io.Copy(rw, req.Body)
}

func reflectDelay(rw http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	_ = req.Body.Close()

	bb := bytes.NewBuffer(body)
	for bb.Len() > 0 {
		chunk := make([]byte, 1024)
		n, readErr := bb.Read(chunk)
		if readErr == io.EOF {
			if n == 0 {
				break
			}
		}
		_, _ = rw.Write(chunk[:n])
		if fl, ok := rw.(http.Flusher); ok {
			fl.Flush()
		}

		time.Sleep(time.Millisecond * 100) // related to backend flush writer default interval
	}
}
