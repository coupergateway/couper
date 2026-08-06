package authz

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"slices"
	"strings"

	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
	"github.com/coupergateway/couper/eval"
	"github.com/coupergateway/couper/eval/buffer"
	"github.com/coupergateway/couper/internal/seetie"
)

const roundTripName = "external_authz"

// External authorization calls out to a policy decision point which decides whether the
// client request is allowed. The decision is in the response body, not in its status code.
type External struct {
	includeTLS          bool
	name                string
	permissionsProperty string
	transport           http.RoundTripper
	url                 string
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

func (e *External) Validate(req *http.Request) error {
	body, err := json.Marshal(newEvaluationRequest(req, e.includeTLS))
	if err != nil {
		return errors.ExternalAuthz.Label(e.name).With(err)
	}

	outreq, err := http.NewRequest(http.MethodPost, e.url, nil)
	if err != nil {
		return errors.ExternalAuthz.Label(e.name).With(err)
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
		return errors.ExternalAuthz.Label(e.name).With(err)
	}
	defer res.Body.Close()

	// An error status reports a problem between Couper and the decision point, not a denied
	// client. Couper must copy nothing from such a response: a 401 says that Couper failed to
	// authenticate to the decision point, and its challenge would mislead the client.
	if res.StatusCode != http.StatusOK {
		return errors.ExternalAuthz.Label(e.name).
			Messagef("unexpected authorization service response status: %d", res.StatusCode)
	}

	evaluation, derr := e.parseResponseBody(res)
	if derr != nil {
		return derr
	}

	e.storeContext(req, evaluationContext(evaluation, res.Header))

	if evaluation.Decision == nil {
		return errors.ExternalAuthz.Label(e.name).Message("missing decision in authorization service response")
	}

	if !*evaluation.Decision {
		return e.deny(evaluation.Context)
	}

	return e.grantPermissions(req, evaluation.Context)
}

// deny maps a decision of false onto an error type. A challenge in the response context
// means invalid credentials: it tells the client how to authenticate, for example with an
// RFC 9728 resource_metadata pointer. Without a challenge the denial is an authorization
// decision, and new credentials would not help the client.
func (e *External) deny(evalContext map[string]interface{}) error {
	if challenge, _ := evalContext[wwwAuthenticateProperty].(string); challenge != "" {
		return errors.ExternalAuthzInvalidCredentials.Label(e.name).Message("invalid credentials")
	}

	return errors.ExternalAuthzInsufficientPermissions.Label(e.name).Message("insufficient permissions")
}

// parseResponseBody reads the access evaluation response. A malformed body denies the
// request: to drop data that a permission check relies on would fail open.
func (e *External) parseResponseBody(res *http.Response) (evaluationResponse, error) {
	var evaluation evaluationResponse

	mediaType, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return evaluation, nil
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return evaluation, errors.ExternalAuthz.Label(e.name).With(err)
	}
	if len(raw) == 0 {
		return evaluation, nil
	}

	if err = json.Unmarshal(raw, &evaluation); err != nil {
		return evaluation, errors.ExternalAuthz.Label(e.name).
			Message("unexpected authorization service response body").With(err)
	}

	return evaluation, nil
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

// evaluationContext flattens the response context for request.context.<label>. Couper adds
// the decision and the callout response headers (lower-cased names, first value, like
// request.headers). Both shadow a response context property of the same name.
func evaluationContext(evaluation evaluationResponse, header http.Header) map[string]interface{} {
	ctx := map[string]interface{}{}
	for name, value := range evaluation.Context {
		ctx[name] = value
	}

	if evaluation.Decision != nil {
		ctx["decision"] = *evaluation.Decision
	}
	ctx["headers"] = seetie.HeaderToMap(header)

	return ctx
}

// grantPermissions appends the permissions from the configured response context property to
// the request's granted permissions with the same value semantics as the jwt block's
// permissions_claim: a space-separated string or a list of strings.
func (e *External) grantPermissions(req *http.Request, evalContext map[string]interface{}) error {
	if e.permissionsProperty == "" {
		return nil
	}

	value, exists := evalContext[e.permissionsProperty]
	if !exists {
		// A configured permissions property expresses a contract with the authorization
		// service; its absence on an allow is a broken service, not an empty grant —
		// failing loudly beats a puzzling 403 at required_permission.
		return errors.ExternalAuthz.Label(e.name).
			Messagef("missing %s permissions property in authorization service response context", e.permissionsProperty)
	}

	invalidErr := func() error {
		return errors.ExternalAuthz.Label(e.name).
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
