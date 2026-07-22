package server_test

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server"
)

func TestExternalAuthz_Callout(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/01_couper.hcl", helper)
	defer shutdown()

	for _, tc := range []struct {
		name          string
		authorization string
		expStatus     int
		expErrorType  string
	}{
		{"valid credentials", "Bearer valid", http.StatusNoContent, ""},
		{"missing credentials", "", http.StatusUnauthorized, "external_authz_invalid_credentials"},
		{"insufficient permissions", "Bearer forbidden", http.StatusForbidden, "external_authz_insufficient_permissions"},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
			helper.Must(err)
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}

			res, err := client.Do(req)
			helper.Must(err)
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()

			if res.StatusCode != tc.expStatus {
				st.Errorf("expected status %d, got: %d", tc.expStatus, res.StatusCode)
			}

			if tc.expErrorType == "" {
				return
			}

			var loggedType string
			for _, entry := range hook.AllEntries() {
				if errorType, ok := entry.Data["error_type"].(string); ok && entry.Data["port"] == "8080" {
					loggedType = errorType
				}
			}
			if loggedType != tc.expErrorType {
				st.Errorf("expected logged error_type %q, got: %q", tc.expErrorType, loggedType)
			}
		})
	}
}

func TestExternalAuthz_ContextPropagation(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/03_couper.hcl", helper)
	defer shutdown()
	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got: %d", http.StatusOK, res.StatusCode)
	}

	if sub := res.Header.Get("X-Authz-Sub"); sub != "clark.kent" {
		t.Errorf("expected authz context sub header, got: %q", sub)
	}

	var body map[string]interface{}
	helper.Must(json.Unmarshal(resBytes, &body))

	if body["sub"] != "clark.kent" {
		t.Errorf("unexpected sub: %v", body["sub"])
	}
	roles, _ := body["roles"].([]interface{})
	if len(roles) != 2 || roles[0] != "reporter" || roles[1] != "hero" {
		t.Errorf("unexpected roles: %v", body["roles"])
	}
}

func TestExternalAuthz_ResponseHeaders(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/04_couper.hcl", helper)
	defer shutdown()
	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
	helper.Must(err)
	req.Header.Set("X-Resolved-Identity", "spoofed")

	res, err := client.Do(req)
	helper.Must(err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got: %d", http.StatusOK, res.StatusCode)
	}

	// The callout's response header is exposed in request.context and wins over the
	// client-provided value; a header the service did not return is empty.
	if identity := res.Header.Get("X-Identity"); identity != "clark.kent" {
		t.Errorf("expected resolved identity from the callout, got: %q", identity)
	}
	if evil := res.Header.Get("X-Evil"); evil != "" {
		t.Errorf("expected no value for an unset callout header, got: %q", evil)
	}
}

func TestExternalAuthz_HTTP2Callout(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	var mu sync.Mutex
	var calloutProtos, calloutConns []string

	authzService := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		mu.Lock()
		calloutProtos = append(calloutProtos, req.Proto)
		calloutConns = append(calloutConns, req.RemoteAddr)
		mu.Unlock()

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{"proto": req.Proto})
	}))
	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)
	authzService.TLS = &tls.Config{Certificates: []tls.Certificate{*selfSigned.Server}}
	authzService.EnableHTTP2 = true
	authzService.StartTLS()
	defer authzService.Close()

	shutdown, hook, err := newCouperWithTemplate("testdata/external_authz/05_couper.hcl", helper,
		map[string]interface{}{"origin": authzService.URL, "ca": string(selfSigned.CACertificate.Certificate)})
	helper.Must(err)
	defer shutdown()
	hook.Reset()

	for range 2 {
		req, rerr := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
		helper.Must(rerr)

		res, derr := client.Do(req)
		helper.Must(derr)
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got: %d", http.StatusOK, res.StatusCode)
		}
		if proto := res.Header.Get("X-Authz-Proto"); proto != "HTTP/2.0" {
			t.Fatalf("expected authz context proto HTTP/2.0, got: %q", proto)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(calloutProtos) != 2 {
		t.Fatalf("expected 2 callouts, got: %d", len(calloutProtos))
	}
	for _, proto := range calloutProtos {
		if proto != "HTTP/2.0" {
			t.Errorf("expected HTTP/2.0 callout, got: %q", proto)
		}
	}
	if calloutConns[0] != calloutConns[1] {
		t.Errorf("expected callouts to reuse one connection, got: %v", calloutConns)
	}
}

func TestExternalAuthz_MTLSClientCertificate(t *testing.T) {
	helper := test.New(t)

	selfSigned, err := server.NewCertificate(time.Minute, nil, nil)
	helper.Must(err)

	var mu sync.Mutex
	var calloutBody []byte
	authzService := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		mu.Lock()
		calloutBody = body
		mu.Unlock()
		rw.WriteHeader(http.StatusOK)
	}))
	defer authzService.Close()

	shutdown, _, err := newCouperWithTemplate("testdata/external_authz/09_couper.hcl", helper, map[string]interface{}{
		"origin":     authzService.URL,
		"publicKey":  string(selfSigned.ServerCertificate.Certificate), // PEM
		"privateKey": string(selfSigned.ServerCertificate.PrivateKey),  // PEM
		"clientCA":   string(selfSigned.CACertificate.Certificate),     // PEM
	})
	helper.Must(err)
	defer shutdown()

	pool := x509.NewCertPool()
	pool.AddCert(selfSigned.CA.Leaf)
	client := test.NewHTTPSClient(&tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{*selfSigned.Client},
	})

	req, err := http.NewRequest(http.MethodGet, "https://localhost:4443/protected", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status %d, got: %d", http.StatusNoContent, res.StatusCode)
	}

	mu.Lock()
	defer mu.Unlock()

	var sent struct {
		MetadataTLS *struct {
			ClientCertificate *struct {
				FingerprintSHA256 string `json:"fingerprint_sha256"`
				SerialNumber      string `json:"serial_number"`
				Subject           string `json:"subject"`
			} `json:"client_certificate"`
		} `json:"metadata_tls"`
	}
	helper.Must(json.Unmarshal(calloutBody, &sent))

	if sent.MetadataTLS == nil || sent.MetadataTLS.ClientCertificate == nil {
		t.Fatalf("expected client_certificate in callout metadata_tls, got: %s", calloutBody)
	}

	// The authorization service must see the exact client certificate Couper terminated,
	// keyed on the fields an mTLS decision relies on.
	leaf := selfSigned.Client.Leaf
	cert := sent.MetadataTLS.ClientCertificate
	if cert.SerialNumber != leaf.SerialNumber.Text(16) {
		t.Errorf("expected serial_number %q, got: %q", leaf.SerialNumber.Text(16), cert.SerialNumber)
	}
	sum := sha256.Sum256(leaf.Raw)
	if cert.FingerprintSHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("expected fingerprint_sha256 %q, got: %q", hex.EncodeToString(sum[:]), cert.FingerprintSHA256)
	}
	if cert.Subject != leaf.Subject.String() {
		t.Errorf("expected subject %q, got: %q", leaf.Subject.String(), cert.Subject)
	}
}

func TestExternalAuthz_PermissionsClaim(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/06_couper.hcl", helper)
	defer shutdown()

	for _, tc := range []struct {
		name          string
		authorization string
		expStatus     int
		expErrorType  string
	}{
		{"granted permission", "Bearer reader", http.StatusNoContent, ""},
		{"missing permission", "Bearer nobody", http.StatusForbidden, "insufficient_permissions"},
	} {
		t.Run(tc.name, func(st *testing.T) {
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
			helper.Must(err)
			req.Header.Set("Authorization", tc.authorization)

			res, err := client.Do(req)
			helper.Must(err)
			_, _ = io.Copy(io.Discard, res.Body)
			_ = res.Body.Close()

			if res.StatusCode != tc.expStatus {
				st.Errorf("expected status %d, got: %d", tc.expStatus, res.StatusCode)
			}

			if tc.expErrorType == "" {
				return
			}

			var loggedType string
			for _, entry := range hook.AllEntries() {
				if errorType, ok := entry.Data["error_type"].(string); ok && entry.Data["port"] == "8080" {
					loggedType = errorType
				}
			}
			if loggedType != tc.expErrorType {
				st.Errorf("expected logged error_type %q, got: %q", tc.expErrorType, loggedType)
			}
		})
	}
}

func TestExternalAuthz_H2CCallout(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	var mu sync.Mutex
	var calloutProtos, calloutConns []string

	h2s := &http2.Server{}
	authzService := httptest.NewServer(h2c.NewHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		mu.Lock()
		calloutProtos = append(calloutProtos, req.Proto)
		calloutConns = append(calloutConns, req.RemoteAddr)
		mu.Unlock()

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{"proto": req.Proto})
	}), h2s))
	defer authzService.Close()

	shutdown, hook, err := newCouperWithTemplate("testdata/external_authz/08_couper.hcl", helper,
		map[string]interface{}{"origin": authzService.URL})
	helper.Must(err)
	defer shutdown()
	hook.Reset()

	for range 2 {
		req, rerr := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
		helper.Must(rerr)

		res, derr := client.Do(req)
		helper.Must(derr)
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status %d, got: %d", http.StatusOK, res.StatusCode)
		}
		if proto := res.Header.Get("X-Authz-Proto"); proto != "HTTP/2.0" {
			t.Fatalf("expected authz context proto HTTP/2.0, got: %q", proto)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(calloutProtos) != 2 {
		t.Fatalf("expected 2 callouts, got: %d", len(calloutProtos))
	}
	for _, proto := range calloutProtos {
		if proto != "HTTP/2.0" {
			t.Errorf("expected HTTP/2.0 cleartext callout, got: %q", proto)
		}
	}
	if calloutConns[0] != calloutConns[1] {
		t.Errorf("expected callouts to reuse one connection, got: %v", calloutConns)
	}
}

func TestExternalAuthz_ChallengeForwarding(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/07_couper.hcl", helper)
	defer shutdown()
	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
	}

	// No error_handler configured: the authorization service's challenge is
	// forwarded by the default handler.
	expChallenge := `Bearer resource_metadata="http://protected.example/.well-known/oauth-protected-resource/protected"`
	if challenge := res.Header.Get("Www-Authenticate"); challenge != expChallenge {
		t.Errorf("expected forwarded challenge %q, got: %q", expChallenge, challenge)
	}
}

func TestExternalAuthz_ErrorHandler(t *testing.T) {
	client := newClient()
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/external_authz/02_couper.hcl", helper)
	defer shutdown()
	hook.Reset()

	req, err := http.NewRequest(http.MethodGet, "http://protected.local:8080/protected", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
	}

	expChallenge := `Bearer resource_metadata="http://protected.example/.well-known/oauth-protected-resource"`
	if challenge := res.Header.Get("Www-Authenticate"); challenge != expChallenge {
		t.Errorf("expected challenge %q, got: %q", expChallenge, challenge)
	}
}
