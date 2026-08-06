package authz

import (
	"net"
	"net/http"
	"strings"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/internal/seetie"
)

// Entity types of the AuthZEN API gateway mapping.
const (
	subjectTypeAnonymous = "anonymous"
	subjectTypeJWT       = "JWT"
	resourceTypeRoute    = "route"
	resourceTypeURI      = "uri"
)

// wwwAuthenticateProperty is a Couper convention on the free-form response context, not a
// part of the specification. AuthZEN denies a request with a flat decision of false, but an
// OAuth 2.0 protected resource must also tell the client how to authenticate.
const wwwAuthenticateProperty = "www_authenticate"

// evaluationRequest is an AuthZEN Authorization API 1.0 access evaluation request.
// The subject, the action and the resource are mandatory.
type evaluationRequest struct {
	Subject  entity                 `json:"subject"`
	Action   action                 `json:"action"`
	Resource entity                 `json:"resource"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

type entity struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

type action struct {
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// evaluationResponse is an AuthZEN access evaluation response. The pointer separates an
// absent decision from an explicit false: a decision point that answers 200 without a
// decision has authorized nothing.
type evaluationResponse struct {
	Decision *bool                  `json:"decision"`
	Context  map[string]interface{} `json:"context"`
}

func newEvaluationRequest(req *http.Request, includeTLS bool) evaluationRequest {
	evalReq := evaluationRequest{
		Subject:  newSubject(req.Header),
		Action:   action{Name: req.Method},
		Resource: newResource(req),
		Context: map[string]interface{}{
			"headers": seetie.HeaderToMap(req.Header),
		},
	}

	if includeTLS {
		if meta := newMetadataTLS(req.TLS); meta != nil {
			evalReq.Context["tls"] = meta
		}
	}

	return evalReq
}

// newSubject names the raw bearer token as the subject. Couper does not validate the
// credential, the decision point does. A request without a bearer token stays anonymous,
// because the subject type and the subject id are mandatory; its credential is still in
// the headers, where the decision point reads it.
func newSubject(header http.Header) entity {
	if token := bearerToken(header); token != "" {
		return entity{Type: subjectTypeJWT, ID: token}
	}

	return entity{Type: subjectTypeAnonymous, ID: subjectTypeAnonymous}
}

// newResource names the matched route, because a policy applies to the route and not to a
// single request path. Without a route, for example in front of a file server, the request
// path is the best identifier that Couper has.
func newResource(req *http.Request) entity {
	properties := map[string]interface{}{
		"hostname": req.URL.Hostname(),
		"ip":       clientIP(req.RemoteAddr),
		"path":     req.URL.Path,
		"scheme":   req.URL.Scheme,
		"uri":      req.URL.String(),
	}

	if query := req.URL.Query(); len(query) > 0 {
		properties["query"] = map[string][]string(query)
	}

	if params, ok := req.Context().Value(request.PathParams).(request.PathParameter); ok && len(params) > 0 {
		properties["params"] = map[string]interface{}(params)
	}

	pattern, _ := req.Context().Value(request.RoutePattern).(string)
	if pattern == "" {
		return entity{Type: resourceTypeURI, ID: req.URL.Path, Properties: properties}
	}

	properties["route"] = pattern

	return entity{Type: resourceTypeRoute, ID: pattern, Properties: properties}
}

// bearerToken is a local copy of the unexported token lookup in the accesscontrol package.
// An import of that package would pull oauth2, oidc, backend and logging into this leaf.
func bearerToken(header http.Header) string {
	const prefix = "bearer "

	authorization := header.Get("Authorization")
	if len(authorization) <= len(prefix) || !strings.EqualFold(authorization[:len(prefix)], prefix) {
		return ""
	}

	return strings.TrimSpace(authorization[len(prefix):])
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return host
}
