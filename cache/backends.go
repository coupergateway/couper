package cache

import (
	"net/http"
	"sync"
)

type StaticBackendsStorage struct {
	db map[string]http.RoundTripper
	mu sync.RWMutex
}

var StaticBackends = &StaticBackendsStorage{
	db: make(map[string]http.RoundTripper),
}

func (s *StaticBackendsStorage) Get(key string) http.RoundTripper {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.db[key]; ok {
		return v
	}

	return nil
}

func (s *StaticBackendsStorage) GetAll() []http.RoundTripper {
	var backends []http.RoundTripper

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, backend := range s.db {
		backends = append(backends, backend)
	}

	return backends
}

func (s *StaticBackendsStorage) Reset() {
	s.mu.Lock()

	s.db = make(map[string]http.RoundTripper)

	s.mu.Unlock()
}

func (s *StaticBackendsStorage) Set(key string, backend http.RoundTripper) {
	s.mu.Lock()

	s.db[key] = backend

	s.mu.Unlock()
}
