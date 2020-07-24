package handler

import (
	"net/http"
)

var (
	_ http.Handler = &Selector{}
	_ Selectable   = &Selector{}
)

type Selectable interface {
	HasResponse(req *http.Request) bool
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
	if a, ok := s.first.(Selectable); ok && a.HasResponse(req) {
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

func (s *Selector) HasResponse(req *http.Request) bool {
	return true
}

func (s *Selector) String() string {
	return s.name
}
