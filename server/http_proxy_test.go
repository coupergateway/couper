package server_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
)

func TestHTTPProxy_Stream(t *testing.T) {
	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/proxy/01_couper.hcl", helper)
	defer shutdown()

	randomBytes := make([]byte, 64*1024) // doubled amount of the proxy byte buffer (32k)
	_, err := rand.Read(randomBytes)
	helper.Must(err)

	outreq, err := http.NewRequest(http.MethodPost, "http://stream.me:8080/", bytes.NewBuffer(randomBytes))
	helper.Must(err)

	client := newClient()
	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d", res.StatusCode)
	}

	if ct := res.Header.Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("expected CT to be 'application/octet-stream', got: %s", ct)
	}

	readBodyStart := time.Now()
	totalBytes := 0
	for {
		bf := make([]byte, 1024)
		n, rerr := res.Body.Read(bf)
		totalBytes += n
		if rerr == io.EOF {
			break
		}
	}
	readBodyTotal := time.Since(readBodyStart)
	helper.Must(res.Body.Close())

	if totalBytes != 65536 {
		t.Errorf("expected 64k bytes, got: %d", totalBytes)
	}

	// backend delays...
	if readBodyTotal < time.Second*6 {
		t.Errorf("expected slower read times with delayed streaming, got a total time of: %s, expected more than 6s", readBodyTotal)
	}
}
