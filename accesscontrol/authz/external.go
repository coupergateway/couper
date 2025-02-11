package authz

import (
	"net/http"

	"github.com/coupergateway/couper/errors"
)

type DestinationRoundTripper interface {
	http.RoundTripper
	GetDestination() string
}

// External authorization calls the configured origin with customized request context.
// The origin must respond with 200 OK to have a valid client request.
type External struct {
	origin             DestinationRoundTripper
	includeMetadataTLS bool
	// TODO
	// conf for who to what
	// params, header or body or both
	// pass certificate
}

//func (c *External) Prepare(backendFunc config.PrepareBackendFunc) error {
//	//TODO implement me
//	panic("implement me")
//}

type clientRequest struct {
	Method string
	URL    string
	Header http.Header
}

type authContext struct {
	Source        any           // previous hop
	Destination   any           // target backend (origin)
	ClientRequest clientRequest // simplified form / serialized
	Route         any
	Metadata      any // user / hcl provided
	MetadataTLS   any // tls conn infos / opt in
}

func NewExternal(origin DestinationRoundTripper, includeMetadataTLS bool) (*External, error) {
	return &External{
		origin:             origin,
		includeMetadataTLS: includeMetadataTLS,
	}, nil
}

func (c *External) Validate(req *http.Request) error {
	if c.origin == nil {
		return errors.AccessControl.Message("origin required")
	}
	//TODO implement me
	panic("implement me")
}
