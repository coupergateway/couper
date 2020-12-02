package test

import "net/http"

// NewHTTPClient creates a new <http.Client> object.
func NewHTTPClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			DisableCompression: true,
		},
	}
	return client
}
