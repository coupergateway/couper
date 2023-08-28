package server_test

import (
	"bytes"
	"crypto/rand"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
)

func TestHTTPProxy_Stream(t *testing.T) {
	helper := test.New(t)

	shutdown, hook := newCouper("testdata/integration/proxy/01_couper.hcl", helper)
	defer shutdown()

	randomBytes := make([]byte, 64*1024) // doubled amount of the proxy byte buffer (32k)
	_, err := rand.Read(randomBytes)
	helper.Must(err)

	outreq, err := http.NewRequest(http.MethodPost, "http://stream.me:8080/", bytes.NewBuffer(randomBytes))
	helper.Must(err)

	client := newClient()
	time.Sleep(time.Second)
	for _, e := range hook.AllEntries() {
		t.Log(e.String())
	}

	res, err := client.Do(outreq)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d", res.StatusCode)
	}

	lastRead := time.Now()
	lastReadAv := time.Duration(0)
	totalBytes := 0
	readCount := 0
	for {
		lrd := time.Since(lastRead)
		lastReadAv += lrd
		bf := make([]byte, 1024)
		n, rerr := res.Body.Read(bf)
		totalBytes += n
		if rerr == io.EOF {
			break
		}

		lastRead = time.Now()
		readCount += 1
	}
	helper.Must(res.Body.Close())

	if totalBytes != 65536 {
		t.Errorf("expected 64k bytes, got: %d", totalBytes)
	}

	// lastReadAv is within nanosecond range (<100ns),
	// should be greater than 0.3 millisecond range while streaming since the backend delays the response chunks.
	if lastReadAv/time.Duration(readCount) < time.Nanosecond*300 {
		t.Errorf("expected slower read times with delayed streaming, got an average of: %s", lastReadAv/time.Duration(readCount))
	}
	t.Log(lastReadAv / time.Duration(readCount))
}
