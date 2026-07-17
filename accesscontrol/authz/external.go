package authz

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
)

const roundTripName = "authz_external"

// External authorization calls out to a service which decides whether the
// client request is allowed: 200 allows, 401 and 403 map to distinct error types.
type External struct {
	includeTLS bool
	name       string
	transport  http.RoundTripper
	url        string
}

// simplified form of http.Request for serialization
type clientRequest struct {
	Method  string      `json:"method"`
	URL     string      `json:"url"`
	Headers http.Header `json:"headers"`
}

// TLS connection information of the client request, forwarded opt-in
type metadataTLS struct {
	CipherSuite       string             `json:"cipher_suite"`
	ClientCertificate *clientCertificate `json:"client_certificate,omitempty"`
	ServerName        string             `json:"server_name,omitempty"`
	Version           string             `json:"version"`
}

type clientCertificate struct {
	Issuer    string    `json:"issuer"`
	NotAfter  time.Time `json:"not_after"`
	NotBefore time.Time `json:"not_before"`
	Subject   string    `json:"subject"`
}

// context sent to external authorization origin
type authContext struct {
	ClientRequest clientRequest `json:"client_request"`
	MetadataTLS   *metadataTLS  `json:"metadata_tls,omitempty"`
}

func NewExternal(name, calloutURL string, includeTLS bool, transport http.RoundTripper) *External {
	if calloutURL == "" { // destination origin is provided by the backend configuration
		calloutURL = "/"
	}
	return &External{
		includeTLS: includeTLS,
		name:       name,
		transport:  transport,
		url:        calloutURL,
	}
}

func newMetadataTLS(state *tls.ConnectionState) *metadataTLS {
	if state == nil {
		return nil
	}

	meta := &metadataTLS{
		CipherSuite: tls.CipherSuiteName(state.CipherSuite),
		ServerName:  state.ServerName,
		Version:     tls.VersionName(state.Version),
	}

	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		meta.ClientCertificate = &clientCertificate{
			Issuer:    cert.Issuer.String(),
			NotAfter:  cert.NotAfter,
			NotBefore: cert.NotBefore,
			Subject:   cert.Subject.String(),
		}
	}

	return meta
}

func (e *External) Validate(req *http.Request) error {
	authCtx := authContext{
		ClientRequest: clientRequest{
			Method:  req.Method,
			URL:     req.URL.String(),
			Headers: req.Header,
		},
	}
	if e.includeTLS {
		authCtx.MetadataTLS = newMetadataTLS(req.TLS)
	}

	body, err := json.Marshal(authCtx)
	if err != nil {
		return errors.AuthzExternal.Label(e.name).With(err)
	}

	outreq, err := http.NewRequest(http.MethodPost, e.url, nil)
	if err != nil {
		return errors.AuthzExternal.Label(e.name).With(err)
	}

	outreq.Header.Set("Accept", "application/json")
	outreq.Header.Set("Content-Type", "application/json")
	eval.SetBody(outreq, body)

	ctx, cancel := context.WithCancel(context.WithValue(req.Context(), request.RoundTripName, roundTripName))
	defer cancel()

	res, err := e.transport.RoundTrip(outreq.WithContext(ctx))
	if err != nil {
		return errors.AuthzExternal.Label(e.name).With(err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return errors.AuthzExternalInvalidCredentials.Label(e.name).Message("invalid credentials")
	case http.StatusForbidden:
		return errors.AuthzExternalInsufficientPermissions.Label(e.name).Message("insufficient permissions")
	default:
		return errors.AuthzExternal.Label(e.name).Messagef("unexpected authorization service response status: %d", res.StatusCode)
	}
}
