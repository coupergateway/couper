package writer_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"testing"
	"time"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server/writer"
)

func TestGzip_Flush(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		bytes, err := ioutil.ReadFile("gzip.go")
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(err.Error()))
			return
		}
		for _, b := range bytes {
			time.Sleep(time.Millisecond / 2)
			_, _ = rw.Write([]byte{b})
		}
	}))
	defer origin.Close()

	rp := &httputil.ReverseProxy{
		Director:      func(_ *http.Request) {},
		FlushInterval: time.Millisecond * 10,
	}

	clientReq, err := http.NewRequest(http.MethodGet, origin.URL, nil)
	helper.Must(err)
	clientReq.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	gzipWriter := writer.NewGzipWriter(rec, clientReq.Header)
	responseWriter := writer.NewResponseWriter(gzipWriter, "")

	rp.ServeHTTP(responseWriter, clientReq)

	rec.Flush()
	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Error("Expected StatusOK")
	}

	if res.Header.Get("Content-Encoding") != "gzip" {
		t.Error("Expected gzip response")
	}
}
