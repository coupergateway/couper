package authz

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
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

	outCtx := context.WithValue(req.Context(), request.RoundTripName, roundTripName)
	// keep the response body readable with a non default roundtrip name
	outCtx = context.WithValue(outCtx, request.BufferOptions, buffer.Option(buffer.Response))
	ctx, cancel := context.WithCancel(outCtx)
	defer cancel()

	res, err := e.transport.RoundTrip(outreq.WithContext(ctx))
	if err != nil {
		return errors.AuthzExternal.Label(e.name).With(err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		return e.storeContext(req, res)
	case http.StatusUnauthorized:
		return errors.AuthzExternalInvalidCredentials.Label(e.name).Message("invalid credentials")
	case http.StatusForbidden:
		return errors.AuthzExternalInsufficientPermissions.Label(e.name).Message("insufficient permissions")
	default:
		return errors.AuthzExternal.Label(e.name).Messagef("unexpected authorization service response status: %d", res.StatusCode)
	}
}

// storeContext exposes the callout response as request.context.<label>: the properties of a
// JSON object response body, plus the response headers under a "headers" property (lower-cased
// names, first value, matching request.headers). Consumers inject a resolved identity or a
// re-signed internal token upstream by reading these and writing set_request_headers, which
// overwrites client-provided values. A malformed JSON body denies the request, as downstream
// permission checks may rely on it. A body property literally named "headers" is shadowed by
// the response headers.
func (e *External) storeContext(req *http.Request, res *http.Response) error {
	data := map[string]interface{}{}

	mediaType, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if mediaType == "application/json" {
		raw, err := io.ReadAll(res.Body)
		if err != nil {
			return errors.AuthzExternal.Label(e.name).With(err)
		}
		if len(raw) > 0 {
			var body map[string]interface{}
			if err = json.Unmarshal(raw, &body); err != nil {
				return errors.AuthzExternal.Label(e.name).Message("unexpected authorization service response body").With(err)
			}
			for name, value := range body {
				data[name] = value
			}
		}
	}

	data["headers"] = responseHeaders(res.Header)

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[e.name] = data
	*req = *req.WithContext(context.WithValue(ctx, request.AccessControls, acMap))

	return nil
}

// responseHeaders renders callout response headers like request.headers: lower-cased names
// mapped to the first value.
func responseHeaders(header http.Header) map[string]interface{} {
	m := make(map[string]interface{}, len(header))
	for name, values := range header {
		value := ""
		if len(values) > 0 {
			value = values[0]
		}
		m[strings.ToLower(name)] = value
	}
	return m
}
