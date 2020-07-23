package handler

import (
	"net/http"
)

var (
	_ http.Handler = &Selector{}
	_ selectable   = &Selector{}
)

type selectable interface {
	hasResponse(req *http.Request) bool
}

type Selector struct {
	first  http.Handler
	second http.Handler
	name   string
}

func NewSelector(first, second http.Handler) *Selector {
	return &Selector{
		first:  first,
		second: second,
	}
}

func (s *Selector) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if a, ok := s.first.(selectable); ok && a.hasResponse(req) {
		if name, ok := s.first.(interface{ String() string }); ok {
			s.name = name.String()
		}

		s.first.ServeHTTP(rw, req)
		return
	}

	if name, ok := s.second.(interface{ String() string }); ok {
		s.name = name.String()
	}

	s.second.ServeHTTP(rw, req)
}

func (s *Selector) hasResponse(req *http.Request) bool {
	return true
}

func (s *Selector) String() string {
	return s.name
}
