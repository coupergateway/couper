package handler

import (
	"fmt"
	"net/http"

	"go.avenga.cloud/couper/gateway/assets"
)

// ServeError writes asset error content into the response
func ServeError(rw http.ResponseWriter, req *http.Request, status int) {
	var file *assets.AssetFile

	if assets.Assets != nil {
		key := fmt.Sprintf("%d.html", status)

		v, err := assets.Assets.Open(key)
		if err == nil {
			file = v
		}
	}

	if file == nil {
		rw.WriteHeader(status)
		return
	}

	if req.Method != "HEAD" {
		if ct := file.CT(); ct != "" {
			rw.Header().Set("Content-Type", ct)
		}
		rw.Header().Set("Content-Length", file.Size())
	}

	rw.WriteHeader(status)

	// TODO: gzip, br?
	if req.Method != "HEAD" {
		rw.Write(file.Bytes())
	}
}
