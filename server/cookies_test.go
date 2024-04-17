package server_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
)

func TestCookies_IntegrationStrip(t *testing.T) {
	helper := test.New(t)
	seenCh := make(chan struct{}, 1)

	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Set-Cookie", "n=v; Path=path; Secure")
		rw.WriteHeader(http.StatusOK)

		seenCh <- struct{}{}
		close(seenCh)
	}))

	ln, err := net.Listen("tcp4", testProxyAddr[7:])
	helper.Must(err)
	origin.Listener = ln
	origin.Start()
	defer origin.Close()

	confPath := "testdata/settings/01_couper.hcl"
	shutdown, _ := newCouper(confPath, test.New(t))
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080", nil)
	helper.Must(err)

	res, err := newClient().Do(req)
	helper.Must(err)

	if v := res.Header.Get("Set-Cookie"); v != "n=v; Path=path" {
		t.Errorf("Unexpected Set-Cookie header given: %s", v)
	}

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		t.Error("Origin request failed")
	case <-seenCh:
	}
}
