package authz

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/config"
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
	body                *hclsyntax.Body
	evaluatePermissions []string
	includeTLS          bool
	name                string
	permissionsProperty string
	transport           http.RoundTripper
	url                 string
}

func NewExternal(conf *config.ExternalAuthZ, transport http.RoundTripper) *External {
	defaultPath := authzenEvaluationPath
	if len(conf.EvaluatePermissions) > 0 {
		defaultPath = authzenEvaluationsPath
	}

	return &External{
		body:                conf.HCLBody(),
		evaluatePermissions: conf.EvaluatePermissions,
		includeTLS:          conf.IncludeTLS,
		name:                conf.Name,
		permissionsProperty: conf.PermissionsProperty,
		transport:           transport,
		url:                 calloutURL(conf.URL, defaultPath),
	}
}

// calloutURL adds the default AuthZEN access evaluation path to a configured URL with an
// empty or root path, so an origin alone is enough to reach a conformant decision point.
func calloutURL(configured, defaultPath string) string {
	if configured == "" { // the backend configuration provides the origin
		return defaultPath
	}

	parsed, err := url.Parse(configured)
	if err != nil || (parsed.Path != "" && parsed.Path != "/") {
		return configured
	}

	parsed.Path = defaultPath

	return parsed.String()
}

func (e *External) Validate(req *http.Request) error {
	evalReq, err := newEvaluationRequest(req, e.includeTLS, e.body)
	if err != nil {
		return errors.ExternalAuthz.Label(e.name).With(err)
	}

	var payload interface{} = evalReq
	if e.batched() {
		payload = newBatchEvaluationRequest(evalReq, e.evaluatePermissions)
	}

	res, err := e.callout(req, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// An error status reports a problem between Couper and the decision point, not a denied
	// client. Couper must copy nothing from such a response: a 401 says that Couper failed to
	// authenticate to the decision point, and its challenge would mislead the client.
	if res.StatusCode != http.StatusOK {
		return errors.ExternalAuthz.Label(e.name).
			Messagef("unexpected authorization service response status: %d", res.StatusCode)
	}

	if e.batched() {
		return e.consumeBatch(req, res)
	}

	return e.consume(req, res)
}

func (e *External) batched() bool {
	return len(e.evaluatePermissions) > 0
}

func (e *External) callout(req *http.Request, payload interface{}) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.ExternalAuthz.Label(e.name).With(err)
	}

	outreq, err := http.NewRequest(http.MethodPost, e.url, nil)
	if err != nil {
		return nil, errors.ExternalAuthz.Label(e.name).With(err)
	}

	outreq.Header.Set("Accept", "application/json")
	outreq.Header.Set("Content-Type", "application/json")
	// A decision point must echo this identifier, which ties its log to the Couper log.
	if uid, _ := req.Context().Value(request.UID).(string); uid != "" {
		outreq.Header.Set("X-Request-ID", uid)
	}
	eval.SetBody(outreq, body)

	outCtx := context.WithValue(req.Context(), request.RoundTripName, roundTripName)
	// keep the response body readable with a non default roundtrip name
	outCtx = context.WithValue(outCtx, request.BufferOptions, buffer.Option(buffer.Response))
	ctx, cancel := context.WithCancel(outCtx)
	defer cancel()

	res, err := e.transport.RoundTrip(outreq.WithContext(ctx))
	if err != nil {
		return nil, errors.ExternalAuthz.Label(e.name).With(err)
	}

	return res, nil
}

func (e *External) consume(req *http.Request, res *http.Response) error {
	var evaluation evaluationResponse
	if err := e.decodeResponseBody(res, &evaluation); err != nil {
		return err
	}

	e.storeContext(req, evaluationContext(evaluation, res.Header))

	if err := e.enforce(evaluation); err != nil {
		return err
	}

	return e.grantConfiguredPermissions(req, evaluation.Context)
}

// consumeBatch applies the boxcarred decisions: the first evaluation answers the client
// request, every other one answers whether the subject may do a candidate permission.
func (e *External) consumeBatch(req *http.Request, res *http.Response) error {
	var batch batchEvaluationResponse
	if err := e.decodeResponseBody(res, &batch); err != nil {
		return err
	}

	if expected := len(e.evaluatePermissions) + 1; len(batch.Evaluations) != expected {
		// execute_all answers every question. A short array leaves permissions unresolved,
		// which would surface as a puzzling 403 at required_permission.
		return errors.ExternalAuthz.Label(e.name).
			Messagef("expected %d evaluations in authorization service response, got: %d",
				expected, len(batch.Evaluations))
	}

	evaluation := batch.Evaluations[0]
	e.storeContext(req, evaluationContext(evaluation, res.Header))

	if err := e.enforce(evaluation); err != nil {
		return err
	}

	var permissions []string
	for i, permission := range e.evaluatePermissions {
		if granted := batch.Evaluations[i+1].Decision; granted != nil && *granted {
			permissions = append(permissions, permission)
		}
	}
	grantPermissions(req, permissions)

	return nil
}

func (e *External) enforce(evaluation evaluationResponse) error {
	if evaluation.Decision == nil {
		return errors.ExternalAuthz.Label(e.name).Message("missing decision in authorization service response")
	}

	if !*evaluation.Decision {
		return e.deny(evaluation.Context)
	}

	return nil
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

// decodeResponseBody reads the access evaluation response into target. A malformed body
// denies the request: to drop data that a permission check relies on would fail open. A
// response without a JSON body leaves target empty, which denies.
func (e *External) decodeResponseBody(res *http.Response, target interface{}) error {
	mediaType, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if mediaType != "application/json" {
		return nil
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.ExternalAuthz.Label(e.name).With(err)
	}
	if len(raw) == 0 {
		return nil
	}

	if err = json.Unmarshal(raw, target); err != nil {
		return errors.ExternalAuthz.Label(e.name).
			Message("unexpected authorization service response body").With(err)
	}

	return nil
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

	// A missing decision denies, so false is the truthful value.
	ctx["decision"] = evaluation.Decision != nil && *evaluation.Decision
	ctx["headers"] = seetie.HeaderToMap(header)

	return ctx
}

// grantConfiguredPermissions appends the permissions from the configured response context
// property to the request's granted permissions with the same value semantics as the jwt
// block's permissions_claim: a space-separated string or a list of strings.
func (e *External) grantConfiguredPermissions(req *http.Request, evalContext map[string]interface{}) error {
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

	grantPermissions(req, permissions)

	return nil
}

// grantPermissions adds permissions the request does not carry yet, so a preceding access
// control keeps what it granted.
func grantPermissions(req *http.Request, permissions []string) {
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
}
