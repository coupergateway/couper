package writer_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"
	"time"

	"github.com/coupergateway/couper/internal/test"
	"github.com/coupergateway/couper/server/writer"
)

func TestGzip_Flush(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fileBytes, err := os.ReadFile("gzip.go")
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(err.Error()))
			return
		}
		for _, b := range fileBytes {
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

func TestGzip_ByPass(t *testing.T) {
	helper := test.New(t)

	expectedBytes, e := os.ReadFile("gzip.go")
	helper.Must(e)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get(writer.AcceptEncodingHeader) == writer.GzipName {
			rw.Header().Set(writer.ContentEncodingHeader, writer.GzipName)
			zw := gzip.NewWriter(rw)
			_, err := zw.Write(expectedBytes)
			helper.Must(err)
			helper.Must(zw.Close())
			return
		}

		_, err := rw.Write(expectedBytes)
		helper.Must(err)
	}))
	defer origin.Close()

	rp := &httputil.ReverseProxy{
		Director: func(_ *http.Request) {},
	}

	for _, testcase := range []struct {
		name     string
		encoding string
	}{
		{name: "with ae gzip", encoding: "gzip"},
		{name: "without ae gzip", encoding: ""},
	} {
		t.Run(testcase.name, func(st *testing.T) {
			clientReq, err := http.NewRequest(http.MethodGet, origin.URL, nil)
			helper.Must(err)
			clientReq.Header.Set(writer.AcceptEncodingHeader, testcase.encoding)

			rec := httptest.NewRecorder()
			gzipWriter := writer.NewGzipWriter(rec, clientReq.Header)

			rp.ServeHTTP(gzipWriter, clientReq)

			rec.Flush()
			res := rec.Result()

			if res.StatusCode != http.StatusOK {
				st.Errorf("Want status-code 200, got: %d", res.StatusCode)
			}

			// create gzip reader by expectation and not by content-encoding
			if testcase.encoding == "gzip" {
				gr, err := gzip.NewReader(res.Body)
				if err != nil {
					st.Fatal(err)
				}

				res.Body = gr
			}

			resultBytes, err := io.ReadAll(res.Body)
			if err != nil {
				st.Errorf("read error with Accept-Encoding %q: %v", testcase.encoding, err)
				return
			}

			_ = res.Body.Close()

			if !bytes.Equal(expectedBytes, resultBytes) {
				t.Errorf("Want %d bytes with Accept-Encoding %q, got %d bytes with Content-Encoding: %s",
					len(expectedBytes), testcase.encoding, len(resultBytes), res.Header.Get("Content-Encoding"))
			}

		})
	}
}
