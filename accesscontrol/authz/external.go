package authz

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/internal/seetie"
)

const roundTripName = "authz_external"

// External authorization calls out to a service which decides whether the
// client request is allowed: 200 allows, 401 and 403 map to distinct error types.
type External struct {
	includeTLS          bool
	name                string
	permissionsProperty string
	transport           http.RoundTripper
	url                 string
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

func NewExternal(name, calloutURL string, includeTLS bool, permissionsProperty string, transport http.RoundTripper) *External {
	if calloutURL == "" { // destination origin is provided by the backend configuration
		calloutURL = "/"
	}
	return &External{
		includeTLS:          includeTLS,
		name:                name,
		permissionsProperty: permissionsProperty,
		transport:           transport,
		url:                 calloutURL,
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
		data, derr := e.parseResponseBody(res)
		if derr != nil {
			return derr
		}
		e.storeContext(req, withResponseHeaders(data, res.Header))
		return e.grantPermissions(req, data)
	case http.StatusUnauthorized:
		return errors.AuthzExternalInvalidCredentials.Label(e.name).Message("invalid credentials")
	case http.StatusForbidden:
		return errors.AuthzExternalInsufficientPermissions.Label(e.name).Message("insufficient permissions")
	default:
		return errors.AuthzExternal.Label(e.name).Messagef("unexpected authorization service response status: %d", res.StatusCode)
	}
}

// parseResponseBody reads a JSON object response body. A malformed body denies
// the request: silently dropping data which downstream permission checks may
// rely on would fail open.
func (e *External) parseResponseBody(res *http.Response) (map[string]interface{}, error) {
	mediaType, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return nil, nil
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.AuthzExternal.Label(e.name).With(err)
	}
	if len(raw) == 0 {
		return nil, nil
	}

	var data map[string]interface{}
	if err = json.Unmarshal(raw, &data); err != nil {
		return nil, errors.AuthzExternal.Label(e.name).Message("unexpected authorization service response body").With(err)
	}

	return data, nil
}

// storeContext exposes the response data as request.context.<label>.
func (e *External) storeContext(req *http.Request, data map[string]interface{}) {
	if data == nil {
		return
	}

	ctx := req.Context()
	acMap, ok := ctx.Value(request.AccessControls).(map[string]interface{})
	if !ok {
		acMap = make(map[string]interface{})
	}
	acMap[e.name] = data
	*req = *req.WithContext(context.WithValue(ctx, request.AccessControls, acMap))
}

// withResponseHeaders adds the callout response headers to the context data under "headers"
// (lower-cased names, first value, like request.headers), without polluting the body map used
// for permissions. A body property literally named "headers" is shadowed.
func withResponseHeaders(data map[string]interface{}, header http.Header) map[string]interface{} {
	ctx := map[string]interface{}{}
	for name, value := range data {
		ctx[name] = value
	}
	ctx["headers"] = seetie.HeaderToMap(header)
	return ctx
}

// grantPermissions appends the permissions from the configured response body
// property to the request's granted permissions with the same value semantics
// as the jwt block's permissions_claim: a space-separated string or a list of strings.
func (e *External) grantPermissions(req *http.Request, data map[string]interface{}) error {
	if e.permissionsProperty == "" {
		return nil
	}

	value, exists := data[e.permissionsProperty]
	if !exists {
		// A configured permissions property expresses a contract with the authorization
		// service; its absence on an allow is a broken service, not an empty grant —
		// failing loudly beats a puzzling 403 at required_permission.
		return errors.AuthzExternal.Label(e.name).
			Messagef("missing %s permissions property in authorization service response", e.permissionsProperty)
	}

	invalidErr := func() error {
		return errors.AuthzExternal.Label(e.name).
			Messagef("invalid %s permissions value: %#v", e.permissionsProperty, value)
	}

	var permissions []string
	switch v := value.(type) {
	case string:
		permissions = strings.Split(v, " ")
	case []interface{}:
		for _, entry := range v {
			p, ok := entry.(string)
			if !ok {
				return invalidErr()
			}
			permissions = append(permissions, p)
		}
	default:
		return invalidErr()
	}

	ctx := req.Context()
	granted, _ := ctx.Value(request.GrantedPermissions).([]string)
	for _, p := range permissions {
		p = strings.TrimSpace(p)
		if p == "" || slices.Contains(granted, p) {
			continue
		}
		granted = append(granted, p)
	}
	*req = *req.WithContext(context.WithValue(ctx, request.GrantedPermissions, granted))

	return nil
}
