package authz

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/internal/seetie"
)

// authzenEvaluationPath is the default path of the AuthZEN access evaluation endpoint.
const authzenEvaluationPath = "/access/v1/evaluation"

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

func newEvaluationRequest(req *http.Request, includeTLS bool, body *hclsyntax.Body) (evaluationRequest, error) {
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

	return applyOverrides(evalReq, req, body)
}

// applyOverrides lets the configuration replace the subject, the action and the resource,
// because each is a closed record with mandatory members and a partial merge would make a
// confusing hybrid. The context is an open bag, and the default members are additive, so a
// configured context merges over them.
func applyOverrides(evalReq evaluationRequest, req *http.Request, body *hclsyntax.Body) (evaluationRequest, error) {
	hclCtx := eval.ContextFromRequest(req).HCLContext()

	if object, configured, err := configuredObject(hclCtx, body, "subject"); err != nil {
		return evalReq, err
	} else if configured {
		if evalReq.Subject, err = newEntity("subject", object); err != nil {
			return evalReq, err
		}
	}

	if object, configured, err := configuredObject(hclCtx, body, "resource"); err != nil {
		return evalReq, err
	} else if configured {
		if evalReq.Resource, err = newEntity("resource", object); err != nil {
			return evalReq, err
		}
	}

	if object, configured, err := configuredObject(hclCtx, body, "action"); err != nil {
		return evalReq, err
	} else if configured {
		name := stringProperty(object, "name")
		if name == "" {
			return evalReq, fmt.Errorf("action requires a name")
		}
		evalReq.Action = action{Name: name, Properties: mapProperty(object, "properties")}
	}

	object, _, err := configuredObject(hclCtx, body, "context")
	if err != nil {
		return evalReq, err
	}
	for name, value := range object {
		evalReq.Context[name] = value
	}

	return evalReq, nil
}

func configuredObject(hclCtx *hcl.EvalContext, body *hclsyntax.Body, name string) (map[string]interface{}, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	if _, exists := body.Attributes[name]; !exists {
		return nil, false, nil
	}

	value, err := eval.ValueFromBodyAttribute(hclCtx, body, name)
	if err != nil {
		return nil, true, err
	}

	// seetie.ValueToMap panics on other known types.
	if t := value.Type(); value.IsKnown() && !value.IsNull() && !t.IsObjectType() && !t.IsMapType() {
		return nil, true, fmt.Errorf("%s must be an object", name)
	}

	return seetie.ValueToMap(value), true, nil
}

// newEntity builds a subject or a resource from a configured object. AuthZEN makes the type
// and the id mandatory, so an empty value denies the request: to ask a decision point about
// an unnamed subject would invite an accidental allow.
func newEntity(kind string, object map[string]interface{}) (entity, error) {
	result := entity{
		Type:       stringProperty(object, "type"),
		ID:         stringProperty(object, "id"),
		Properties: mapProperty(object, "properties"),
	}

	if result.Type == "" || result.ID == "" {
		return result, fmt.Errorf("%s requires a type and an id", kind)
	}

	return result, nil
}

func stringProperty(object map[string]interface{}, name string) string {
	value, _ := object[name].(string)
	return strings.TrimSpace(value)
}

func mapProperty(object map[string]interface{}, name string) map[string]interface{} {
	value, _ := object[name].(map[string]interface{})
	return value
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
