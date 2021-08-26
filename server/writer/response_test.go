package writer_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avenga/couper/internal/test"
	"github.com/avenga/couper/server/writer"
)

func TestResponse_ChunkWrite(t *testing.T) {
	helper := test.New(t)

	rec := httptest.NewRecorder()
	w := writer.NewResponseWriter(rec, "")

	content := []byte("HTTP/1.1 404 Not Found\r\n\r\nBody")
	for i := 0; i < len(content); i++ {
		_, err := w.Write([]byte{content[i]})
		helper.Must(err)
	}

	res := rec.Result()
	if res.StatusCode != http.StatusNotFound || w.StatusCode() != http.StatusNotFound {
		t.Errorf("Want: %d, got: %d", http.StatusNotFound, res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	if !bytes.Equal(b, []byte("Body")) {
		t.Errorf("Expected body content, got: %q", string(b))
	}

	if w.WrittenBytes() != 4 {
		t.Errorf("Expected 4 written bytes, got: %d", w.WrittenBytes())
	}
}

func TestResponse_ProtoWrite(t *testing.T) {
	helper := test.New(t)

	rec := httptest.NewRecorder()
	w := writer.NewResponseWriter(rec, "")

	response := &http.Response{
		StatusCode: http.StatusOK,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: http.Header{
			"Test": []string{"Value"},
		},
		Body:          io.NopCloser(bytes.NewBufferString("testContent")),
		ContentLength: 11,
	}

	helper.Must(response.Write(w))

	res := rec.Result()
	if res.StatusCode != http.StatusOK || w.StatusCode() != http.StatusOK {
		t.Errorf("Want: %d, got: %d", http.StatusOK, res.StatusCode)
	}

	if res.Header.Get("Content-Length") != "11" {
		t.Error("Expected Content-Length header")
	}

	b, err := io.ReadAll(res.Body)
	helper.Must(err)

	if !bytes.Equal(b, []byte("testContent")) {
		t.Errorf("Expected body content, got: %q", string(b))
	}

	if w.WrittenBytes() != 11 {
		t.Errorf("Expected 11 written bytes, got: %d", w.WrittenBytes())
	}
}
