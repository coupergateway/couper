package backend

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

type Spa struct {
	rw     http.ResponseWriter
	status int
	body   []byte
}

func (s *Spa) Header() http.Header {
	return s.rw.Header()
}

func (s *Spa) WriteHeader(status int) {
	s.status = status
	if status != http.StatusNotFound {
		s.rw.WriteHeader(status)
	}
}

func (s *Spa) Write(p []byte) (int, error) {
	if s.status != http.StatusNotFound {
		return s.rw.Write(p)
	}
	s.body = p
	return len(p), nil
}

func wrapHandler(h http.Handler, bf string, spa_paths []string) (http.HandlerFunc, error) {
	dir, _ := os.Getwd()
	bs_content, err := ioutil.ReadFile(path.Join(dir, bf))
	if err != nil {
		return nil, err
	}
	return func(rw http.ResponseWriter, req *http.Request) {
		sh := &Spa{rw: rw}
		h.ServeHTTP(sh, req)
		if sh.status == http.StatusNotFound {
			for _, a := range spa_paths {
				// TODO implement wildcard match
				if req.URL.Path == a {
					rw.Write(bs_content)
					return
				}
			}
			rw.Write(sh.body)
		}
	}, nil
}