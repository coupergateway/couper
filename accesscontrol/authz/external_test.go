package authz_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/coupergateway/couper/accesscontrol/authz"
	"github.com/coupergateway/couper/config"
	"github.com/coupergateway/couper/config/request"
	"github.com/coupergateway/couper/errors"
)

func newTestExternal(name, calloutURL string, includeTLS bool, permissionsProperty string,
	transport http.RoundTripper) *authz.External {
	return newConfiguredExternal(&config.ExternalAuthZ{
		IncludeTLS:          includeTLS,
		Name:                name,
		PermissionsProperty: permissionsProperty,
		URL:                 calloutURL,
	}, transport)
}

func newConfiguredExternal(conf *config.ExternalAuthZ, transport http.RoundTripper) *authz.External {
	if conf.Remain == nil {
		conf.Remain = &hclsyntax.Body{}
	}
	return authz.NewExternal(conf, transport)
}

func bodyFromHCL(t *testing.T, src string) *hclsyntax.Body {
	t.Helper()

	file, diags := hclsyntax.ParseConfig([]byte(src), "test.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags)
	}

	return file.Body.(*hclsyntax.Body)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func respondStatus(status int) roundTripperFunc {
	return func(_ *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.WriteHeader(status)
		return rec.Result(), nil
	}
}

func respondJSONBody(status int, body string) roundTripperFunc {
	return func(_ *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.Header().Set("Content-Type", "application/json")
		rec.WriteHeader(status)
		_, _ = rec.WriteString(body)
		return rec.Result(), nil
	}
}

func respondAllow() roundTripperFunc {
	return respondJSONBody(http.StatusOK, `{"decision": true}`)
}

func TestExternal_Validate_Decision(t *testing.T) {
	for _, tc := range []struct {
		name      string
		responder roundTripperFunc
		expKind   string
	}{
		{"allow", respondAllow(), ""},
		{"deny without a challenge", respondJSONBody(http.StatusOK, `{"decision": false}`),
			"external_authz_insufficient_permissions"},
		{"deny with a challenge", respondJSONBody(http.StatusOK,
			`{"decision": false, "context": {"www_authenticate": "Bearer error=\"invalid_token\""}}`),
			"external_authz_invalid_credentials"},
		{"deny with an empty challenge", respondJSONBody(http.StatusOK,
			`{"decision": false, "context": {"www_authenticate": ""}}`),
			"external_authz_insufficient_permissions"},
		{"missing decision", respondJSONBody(http.StatusOK, `{"context": {}}`), "external_authz"},
		{"decision is no boolean", respondJSONBody(http.StatusOK, `{"decision": "true"}`), "external_authz"},
		{"empty body", respondStatus(http.StatusOK), "external_authz"},
		{"malformed body", respondJSONBody(http.StatusOK, `{`), "external_authz"},
		// A decision point error is a fault between Couper and the decision point, never a
		// statement about the client.
		{"decision point rejects couper", respondStatus(http.StatusUnauthorized), "external_authz"},
		{"decision point forbids couper", respondStatus(http.StatusForbidden), "external_authz"},
		{"decision point fails", respondStatus(http.StatusInternalServerError), "external_authz"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			external := newTestExternal("test_ac", "http://authz.service/check", false, "", tc.responder)

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			err := external.Validate(req)

			if tc.expKind == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}

			cErr, ok := err.(*errors.Error)
			if !ok {
				t.Fatalf("expected *errors.Error, got: %T", err)
			}

			kinds := cErr.Kinds()
			if len(kinds) == 0 || kinds[0] != tc.expKind {
				t.Errorf("expected most specific error kind %q, got: %v", tc.expKind, kinds)
			}
		})
	}
}

func TestExternal_Validate_ProtocolErrorHidesChallenge(t *testing.T) {
	transport := roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		rec.Header().Set("WWW-Authenticate", `Bearer realm="pdp.example"`)
		rec.WriteHeader(http.StatusUnauthorized)
		return rec.Result(), nil
	})

	external := newTestExternal("test_ac", "http://authz.service/check", false, "", transport)
	req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

	if err := external.Validate(req); err == nil {
		t.Fatal("expected an error")
	}

	// The decision point challenges Couper, not the client. Nothing of that response may
	// reach the evaluation context, from where an error handler would forward it.
	acMap, _ := req.Context().Value(request.AccessControls).(map[string]interface{})
	if data, exist := acMap["test_ac"]; exist {
		t.Errorf("expected no access control context, got: %v", data)
	}
}

// captureCallout returns the transport plus pointers to the callout request and its body.
func captureCallout(responder roundTripperFunc) (roundTripperFunc, **http.Request, *[]byte) {
	var calloutReq *http.Request
	var calloutBody []byte

	transport := roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		calloutReq = req
		calloutBody, _ = io.ReadAll(req.Body)
		return responder(req)
	})

	return transport, &calloutReq, &calloutBody
}

func decodeCallout(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()

	var sent map[string]interface{}
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("callout body is no valid json: %v", err)
	}

	return sent
}

func objectAt(t *testing.T, parent map[string]interface{}, key string) map[string]interface{} {
	t.Helper()

	child, _ := parent[key].(map[string]interface{})
	if child == nil {
		t.Fatalf("missing %q object in %v", key, parent)
	}

	return child
}

func TestExternal_Validate_CalloutRequest(t *testing.T) {
	transport, calloutReq, calloutBody := captureCallout(respondAllow())
	external := newTestExternal("test_ac", "http://authz.service/check", false, "", transport)

	req := httptest.NewRequest(http.MethodDelete, "http://client.request/todos/42?a=b", nil)
	req.Header.Set("Authorization", "Bearer my-token")
	req.RemoteAddr = "10.0.0.7:54321"
	ctx := context.WithValue(req.Context(), request.RoutePattern, "/todos/{id}")
	ctx = context.WithValue(ctx, request.PathParams, request.PathParameter{"id": "42"})
	req = req.WithContext(ctx)

	if err := external.Validate(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if (*calloutReq).Method != http.MethodPost {
		t.Errorf("expected POST callout, got: %s", (*calloutReq).Method)
	}
	if (*calloutReq).URL.String() != "http://authz.service/check" {
		t.Errorf("unexpected callout url: %s", (*calloutReq).URL)
	}
	if ct := (*calloutReq).Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("unexpected content type: %q", ct)
	}

	sent := decodeCallout(t, *calloutBody)

	subject := objectAt(t, sent, "subject")
	if subject["type"] != "JWT" {
		t.Errorf("unexpected subject type: %v", subject["type"])
	}
	if subject["id"] != "my-token" {
		t.Errorf("unexpected subject id: %v", subject["id"])
	}

	action := objectAt(t, sent, "action")
	if action["name"] != http.MethodDelete {
		t.Errorf("unexpected action name: %v", action["name"])
	}

	resource := objectAt(t, sent, "resource")
	if resource["type"] != "route" {
		t.Errorf("unexpected resource type: %v", resource["type"])
	}
	if resource["id"] != "/todos/{id}" {
		t.Errorf("unexpected resource id: %v", resource["id"])
	}

	properties := objectAt(t, resource, "properties")
	for key, want := range map[string]string{
		"hostname": "client.request",
		"ip":       "10.0.0.7",
		"path":     "/todos/42",
		"route":    "/todos/{id}",
		"scheme":   "http",
		"uri":      "http://client.request/todos/42?a=b",
	} {
		if properties[key] != want {
			t.Errorf("resource property %q: expected %q, got: %v", key, want, properties[key])
		}
	}
	assertJSONStrings(t, objectAt(t, properties, "query"), "a", []string{"b"})
	if params := objectAt(t, properties, "params"); params["id"] != "42" {
		t.Errorf("unexpected path parameters: %v", params)
	}

	headers := objectAt(t, objectAt(t, sent, "context"), "headers")
	if headers["authorization"] != "Bearer my-token" {
		t.Errorf("unexpected authorization header: %v", headers["authorization"])
	}
}

func TestExternal_Validate_CalloutRequest_Fallbacks(t *testing.T) {
	t.Run("anonymous subject without bearer token", func(t *testing.T) {
		transport, _, calloutBody := captureCallout(respondAllow())
		external := newTestExternal("test_ac", "http://authz.service/check", false, "", transport)

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		req.Header.Set("X-Api-Key", "opaque-key")

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, *calloutBody)
		subject := objectAt(t, sent, "subject")
		if subject["type"] != "anonymous" || subject["id"] != "anonymous" {
			t.Errorf("expected anonymous subject, got: %v", subject)
		}

		// the opaque credential must stay reachable for the decision point
		headers := objectAt(t, objectAt(t, sent, "context"), "headers")
		if headers["x-api-key"] != "opaque-key" {
			t.Errorf("unexpected api key header: %v", headers["x-api-key"])
		}
	})

	t.Run("uri resource without route pattern", func(t *testing.T) {
		transport, _, calloutBody := captureCallout(respondAllow())
		external := newTestExternal("test_ac", "http://authz.service/check", false, "", transport)

		req := httptest.NewRequest(http.MethodGet, "http://client.request/static/logo.svg", nil)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, *calloutBody)
		resource := objectAt(t, sent, "resource")
		if resource["type"] != "uri" {
			t.Errorf("unexpected resource type: %v", resource["type"])
		}
		if resource["id"] != "/static/logo.svg" {
			t.Errorf("unexpected resource id: %v", resource["id"])
		}
		if _, exist := objectAt(t, resource, "properties")["route"]; exist {
			t.Error("expected no route property without a matched route")
		}
	})
}

func TestExternal_Validate_ConfiguredEntities(t *testing.T) {
	newExternal := func(src string, transport http.RoundTripper) *authz.External {
		return newConfiguredExternal(&config.ExternalAuthZ{
			Name:   "test_ac",
			Remain: bodyFromHCL(t, src),
			URL:    "http://authz.service/check",
		}, transport)
	}

	t.Run("subject, action and resource replace their default", func(t *testing.T) {
		transport, _, calloutBody := captureCallout(respondAllow())
		external := newExternal(`
			subject  = { type = "identity", id = "alice@example.com" }
			action   = { name = "can_read" }
			resource = { type = "account", id = "123", properties = { tier = "gold" } }
		`, transport)

		req := httptest.NewRequest(http.MethodDelete, "http://client.request/todos/42", nil)
		req.Header.Set("Authorization", "Bearer my-token")
		req = req.WithContext(context.WithValue(req.Context(), request.RoutePattern, "/todos/{id}"))

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, *calloutBody)

		subject := objectAt(t, sent, "subject")
		if subject["type"] != "identity" || subject["id"] != "alice@example.com" {
			t.Errorf("unexpected subject: %v", subject)
		}
		if action := objectAt(t, sent, "action"); action["name"] != "can_read" {
			t.Errorf("unexpected action: %v", action)
		}

		resource := objectAt(t, sent, "resource")
		if resource["type"] != "account" || resource["id"] != "123" {
			t.Errorf("unexpected resource: %v", resource)
		}
		// the configured object replaces the default, so the route properties are gone
		properties := objectAt(t, resource, "properties")
		if properties["tier"] != "gold" || properties["route"] != nil {
			t.Errorf("unexpected resource properties: %v", properties)
		}
	})

	t.Run("context merges over the defaults", func(t *testing.T) {
		transport, _, calloutBody := captureCallout(respondAllow())
		external := newExternal(`context = { tenant = "acme", headers = "replaced" }`, transport)

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		evalContext := objectAt(t, decodeCallout(t, *calloutBody), "context")
		if evalContext["tenant"] != "acme" {
			t.Errorf("expected the configured tenant, got: %v", evalContext["tenant"])
		}
		if evalContext["headers"] != "replaced" {
			t.Errorf("expected a configured key to win, got: %v", evalContext["headers"])
		}
	})

	t.Run("a non-object value fails closed", func(t *testing.T) {
		for _, src := range []string{
			`subject = "foo"`,
			`subject = 42`,
			`resource = true`,
			`action = ["a"]`,
			`context = []`,
		} {
			external := newExternal(src, respondAllow())

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			if err := external.Validate(req); err == nil {
				t.Errorf("expected an error for %s", src)
			}
		}
	})

	t.Run("an incomplete entity fails closed", func(t *testing.T) {
		for _, src := range []string{
			`subject = { type = "identity", id = "" }`,
			`subject = { type = "identity" }`,
			`resource = { type = "account" }`,
			`action = { name = "" }`,
		} {
			external := newExternal(src, respondAllow())

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			if err := external.Validate(req); err == nil {
				t.Errorf("expected an error for %s", src)
			}
		}
	})
}

func TestExternal_Validate_BatchPermissions(t *testing.T) {
	newBatchExternal := func(transport http.RoundTripper) *authz.External {
		return newConfiguredExternal(&config.ExternalAuthZ{
			EvaluatePermissions: []string{"can_read", "can_write"},
			Name:                "test_ac",
			URL:                 "http://authz.service",
		}, transport)
	}

	t.Run("one callout asks about the request and every permission", func(t *testing.T) {
		transport, calloutReq, calloutBody := captureCallout(respondJSONBody(http.StatusOK,
			`{"evaluations": [{"decision": true}, {"decision": true}, {"decision": false}]}`))
		external := newBatchExternal(transport)

		req := httptest.NewRequest(http.MethodGet, "http://client.request/todos/42", nil)
		req = req.WithContext(context.WithValue(req.Context(), request.RoutePattern, "/todos/{id}"))

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		if u := (*calloutReq).URL.String(); u != "http://authz.service/access/v1/evaluations" {
			t.Errorf("unexpected callout url: %s", u)
		}

		sent := decodeCallout(t, *calloutBody)
		if semantic := objectAt(t, sent, "options")["evaluations_semantic"]; semantic != "execute_all" {
			t.Errorf("unexpected evaluations semantic: %v", semantic)
		}
		// subject, resource and context are the shared defaults of every entry
		if id := objectAt(t, sent, "resource")["id"]; id != "/todos/{id}" {
			t.Errorf("unexpected shared resource: %v", id)
		}

		evaluations, _ := sent["evaluations"].([]interface{})
		if len(evaluations) != 3 {
			t.Fatalf("expected 3 evaluations, got: %v", sent["evaluations"])
		}
		for i, want := range []string{http.MethodGet, "can_read", "can_write"} {
			entry, _ := evaluations[i].(map[string]interface{})
			if name := objectAt(t, entry, "action")["name"]; name != want {
				t.Errorf("evaluation %d: expected action %q, got: %v", i, want, name)
			}
		}

		granted, _ := req.Context().Value(request.GrantedPermissions).([]string)
		if len(granted) != 1 || granted[0] != "can_read" {
			t.Errorf("unexpected granted permissions: %v", granted)
		}
	})

	t.Run("the first evaluation answers the client request", func(t *testing.T) {
		external := newBatchExternal(respondJSONBody(http.StatusOK,
			`{"evaluations": [{"decision": false}, {"decision": true}, {"decision": true}]}`))
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		err := external.Validate(req)
		cErr, ok := err.(*errors.Error)
		if !ok {
			t.Fatalf("expected *errors.Error, got: %T", err)
		}
		if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz_insufficient_permissions" {
			t.Errorf("unexpected error kinds: %v", kinds)
		}
		if granted, _ := req.Context().Value(request.GrantedPermissions).([]string); len(granted) != 0 {
			t.Errorf("expected no granted permissions on a denial, got: %v", granted)
		}
	})

	t.Run("an incomplete evaluations array fails closed", func(t *testing.T) {
		external := newBatchExternal(respondJSONBody(http.StatusOK,
			`{"evaluations": [{"decision": true}, {"decision": true}]}`))
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for an incomplete evaluations array")
		}
	})

	t.Run("an oversized evaluations array fails closed", func(t *testing.T) {
		external := newBatchExternal(respondJSONBody(http.StatusOK,
			`{"evaluations": [{"decision": true}, {"decision": true}, {"decision": true}, {"decision": true}]}`))
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for an oversized evaluations array")
		}
	})
}

func TestExternal_Validate_ContextPropagation(t *testing.T) {
	respondBody := func(contentType, body string) roundTripperFunc {
		return func(_ *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			if contentType != "" {
				rec.Header().Set("Content-Type", contentType)
			}
			rec.WriteHeader(http.StatusOK)
			_, _ = rec.WriteString(body)
			return rec.Result(), nil
		}
	}

	contextData := func(req *http.Request) map[string]interface{} {
		acMap, _ := req.Context().Value(request.AccessControls).(map[string]interface{})
		data, _ := acMap["test_ac"].(map[string]interface{})
		return data
	}

	t.Run("json object response lands in access control context", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"decision":true,"context":{"sub":"clark.kent","roles":["reporter"]}}`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data := contextData(req)
		if data == nil {
			t.Fatal("missing access control context data")
		}
		if data["sub"] != "clark.kent" {
			t.Errorf("unexpected sub: %v", data["sub"])
		}
		roles, _ := data["roles"].([]interface{})
		if len(roles) != 1 || roles[0] != "reporter" {
			t.Errorf("unexpected roles: %v", data["roles"])
		}
	})

	t.Run("response headers are exposed under headers", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
				rec := httptest.NewRecorder()
				rec.Header().Set("Content-Type", "application/json")
				rec.Header().Set("X-Resolved-Identity", "clark.kent")
				rec.WriteHeader(http.StatusOK)
				_, _ = rec.WriteString(`{"decision":true,"context":{"sub":"clark.kent"}}`)
				return rec.Result(), nil
			}))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		headers, _ := contextData(req)["headers"].(map[string]interface{})
		if headers["x-resolved-identity"] != "clark.kent" {
			t.Errorf("expected lower-cased x-resolved-identity, got: %v", headers)
		}
	})

	t.Run("invalid json fails closed", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"decision":`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		err := external.Validate(req)

		cErr, ok := err.(*errors.Error)
		if !ok {
			t.Fatalf("expected *errors.Error, got: %T", err)
		}
		if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
			t.Errorf("expected error kind external_authz, got: %v", kinds)
		}
	})

	t.Run("non-object json fails closed", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `[1,2]`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Error("expected an error for a non-object json response")
		}
	})

	t.Run("a denial still lands in the access control context", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"decision":false,"context":{"reason":"no seat"}}`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error")
		}

		// an error handler reads the reason of the denial from here
		data := contextData(req)
		if data["reason"] != "no seat" {
			t.Errorf("unexpected reason: %v", data["reason"])
		}
		if data["decision"] != false {
			t.Errorf("expected the decision in the context data, got: %v", data["decision"])
		}
	})

	t.Run("a missing decision is exposed as false", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"context":{"reason":"broken"}}`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for a response without a decision")
		}
		if data := contextData(req); data["decision"] != false {
			t.Errorf("expected decision false in context data, got: %v", data["decision"])
		}
	})

	t.Run("the decision shadows a response context property of the same name", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", `{"context":{"decision":true}}`))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for a response without a decision")
		}
		if data := contextData(req); data["decision"] != false {
			t.Errorf("expected the context property shadowed with false, got: %v", data["decision"])
		}
	})

	t.Run("empty body fails closed but exposes headers", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("application/json", ""))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for a response without a decision")
		}
		if _, ok := contextData(req)["headers"]; !ok {
			t.Error("expected headers in context data")
		}
	})

	t.Run("non-json response fails closed", func(t *testing.T) {
		external := newTestExternal("test_ac", "http://authz.service/check", false, "",
			respondBody("text/plain", "OK"))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err == nil {
			t.Fatal("expected an error for a response without a decision")
		}
		data := contextData(req)
		if _, ok := data["headers"]; !ok {
			t.Error("expected headers in context data")
		}
		if data["sub"] != nil {
			t.Errorf("expected no body properties, got: %v", data)
		}
	})
}

func TestExternal_Validate_PermissionsProperty(t *testing.T) {
	respondJSON := func(body string) roundTripperFunc {
		return func(_ *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.Header().Set("Content-Type", "application/json")
			rec.WriteHeader(http.StatusOK)
			_, _ = rec.WriteString(body)
			return rec.Result(), nil
		}
	}

	granted := func(req *http.Request) []string {
		permissions, _ := req.Context().Value(request.GrantedPermissions).([]string)
		return permissions
	}

	newExternal := func(evalContext string) *authz.External {
		body := `{"decision": true, "context": ` + evalContext + `}`
		return newTestExternal("test_ac", "http://authz.service/check", false, "perms", respondJSON(body))
	}

	t.Run("list property grants permissions", func(t *testing.T) {
		external := newExternal(`{"perms":["read","write"]}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 2 || p[0] != "read" || p[1] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("space-separated string grants permissions", func(t *testing.T) {
		external := newExternal(`{"perms":"read write"}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 2 || p[0] != "read" || p[1] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("appends to and dedupes against already granted permissions", func(t *testing.T) {
		external := newExternal(`{"perms":["read","write"]}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		ctx := context.WithValue(req.Context(), request.GrantedPermissions, []string{"admin", "read"})
		req = req.WithContext(ctx)

		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if p := granted(req); len(p) != 3 || p[0] != "admin" || p[1] != "read" || p[2] != "write" {
			t.Errorf("unexpected granted permissions: %v", p)
		}
	})

	t.Run("missing property denies", func(t *testing.T) {
		external := newExternal(`{"sub":"clark.kent"}`)
		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

		err := external.Validate(req)
		if err == nil {
			t.Fatal("expected an error for the missing permissions property")
		}
		logErr, ok := err.(errors.GoError)
		if !ok || !strings.Contains(logErr.LogError(), "missing perms permissions property in authorization service response context") {
			t.Errorf("unexpected error: %v", err)
		}
		if p := granted(req); len(p) != 0 {
			t.Errorf("expected no granted permissions, got: %v", p)
		}
	})

	t.Run("invalid property type fails closed", func(t *testing.T) {
		for _, body := range []string{`{"perms":42}`, `{"perms":["read",42]}`} {
			external := newExternal(body)
			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)

			err := external.Validate(req)
			cErr, ok := err.(*errors.Error)
			if !ok {
				t.Fatalf("expected *errors.Error for %s, got: %T", body, err)
			}
			if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
				t.Errorf("expected error kind external_authz for %s, got: %v", body, kinds)
			}
		}
	})
}

func assertJSONStrings(t *testing.T, obj map[string]interface{}, key string, want []string) {
	t.Helper()
	values, ok := obj[key].([]interface{})
	if !ok || len(values) != len(want) {
		t.Errorf("%s: expected %v, got: %v", key, want, obj[key])
		return
	}
	for i, w := range want {
		if values[i] != w {
			t.Errorf("%s[%d]: expected %q, got: %v", key, i, w, values[i])
		}
	}
}

func TestExternal_Validate_IncludeTLS(t *testing.T) {
	newTLSRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodGet, "https://client.request/protected", nil)
		req.TLS = &tls.ConnectionState{
			Version:     tls.VersionTLS13,
			CipherSuite: tls.TLS_AES_128_GCM_SHA256,
			ServerName:  "client.request",
			PeerCertificates: []*x509.Certificate{{
				Subject:   pkix.Name{CommonName: "my-client"},
				Issuer:    pkix.Name{CommonName: "my-ca"},
				NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				NotAfter:  time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			}},
		}
		return req
	}

	captureBody := func(target *[]byte) roundTripperFunc {
		return func(req *http.Request) (*http.Response, error) {
			*target, _ = io.ReadAll(req.Body)
			return respondAllow()(req)
		}
	}

	t.Run("enabled", func(t *testing.T) {
		var calloutBody []byte
		external := newTestExternal("test_ac", "http://authz.service/check", true, "", captureBody(&calloutBody))

		if err := external.Validate(newTLSRequest()); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, calloutBody)

		metaTLS := objectAt(t, objectAt(t, sent, "context"), "tls")
		if metaTLS["version"] != "TLS 1.3" {
			t.Errorf("unexpected tls version: %v", metaTLS["version"])
		}
		if metaTLS["cipher_suite"] != "TLS_AES_128_GCM_SHA256" {
			t.Errorf("unexpected cipher suite: %v", metaTLS["cipher_suite"])
		}
		if metaTLS["server_name"] != "client.request" {
			t.Errorf("unexpected server name: %v", metaTLS["server_name"])
		}
		cert, _ := metaTLS["client_certificate"].(map[string]interface{})
		if cert == nil {
			t.Fatal("missing client_certificate object")
		}
		if cert["subject"] != "CN=my-client" {
			t.Errorf("unexpected certificate subject: %v", cert["subject"])
		}
		if cert["issuer"] != "CN=my-ca" {
			t.Errorf("unexpected certificate issuer: %v", cert["issuer"])
		}
	})

	t.Run("client certificate identity fields", func(t *testing.T) {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		spiffe, _ := url.Parse("spiffe://example.org/mcp-client")
		template := &x509.Certificate{
			SerialNumber:   big.NewInt(4711),
			Subject:        pkix.Name{CommonName: "mcp-client"},
			NotBefore:      time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			NotAfter:       time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
			DNSNames:       []string{"client.example"},
			EmailAddresses: []string{"mcp@example.org"},
			IPAddresses:    []net.IP{net.ParseIP("10.0.0.7")},
			URIs:           []*url.URL{spiffe},
		}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodGet, "https://client.request/protected", nil)
		req.TLS = &tls.ConnectionState{
			Version:          tls.VersionTLS13,
			CipherSuite:      tls.TLS_AES_128_GCM_SHA256,
			PeerCertificates: []*x509.Certificate{cert},
		}

		var calloutBody []byte
		external := newTestExternal("test_ac", "http://authz.service/check", true, "", captureBody(&calloutBody))
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, calloutBody)
		metaTLS := objectAt(t, objectAt(t, sent, "context"), "tls")
		clientCert := objectAt(t, metaTLS, "client_certificate")

		if clientCert["subject"] != "CN=mcp-client" {
			t.Errorf("unexpected subject: %v", clientCert["subject"])
		}
		if clientCert["serial_number"] != "1267" { // 4711 in hex
			t.Errorf("unexpected serial_number: %v", clientCert["serial_number"])
		}
		sum := sha256.Sum256(der)
		if clientCert["fingerprint_sha256"] != hex.EncodeToString(sum[:]) {
			t.Errorf("unexpected fingerprint_sha256: %v", clientCert["fingerprint_sha256"])
		}
		assertJSONStrings(t, clientCert, "dns_names", []string{"client.example"})
		assertJSONStrings(t, clientCert, "email_addresses", []string{"mcp@example.org"})
		assertJSONStrings(t, clientCert, "ip_addresses", []string{"10.0.0.7"})
		assertJSONStrings(t, clientCert, "uris", []string{"spiffe://example.org/mcp-client"})
	})

	t.Run("enabled without tls connection", func(t *testing.T) {
		var calloutBody []byte
		external := newTestExternal("test_ac", "http://authz.service/check", true, "", captureBody(&calloutBody))

		req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
		if err := external.Validate(req); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, calloutBody)
		if _, exist := objectAt(t, sent, "context")["tls"]; exist {
			t.Error("expected no tls context for a non-tls connection")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		var calloutBody []byte
		external := newTestExternal("test_ac", "http://authz.service/check", false, "", captureBody(&calloutBody))

		if err := external.Validate(newTLSRequest()); err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		sent := decodeCallout(t, calloutBody)
		if _, exist := objectAt(t, sent, "context")["tls"]; exist {
			t.Error("expected no tls context when include_tls is disabled")
		}
	})
}

func TestExternal_Validate_TransportError(t *testing.T) {
	external := newTestExternal("test_ac", "http://authz.service/check", false, "", roundTripperFunc(
		func(_ *http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		}))

	req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
	err := external.Validate(req)

	cErr, ok := err.(*errors.Error)
	if !ok {
		t.Fatalf("expected *errors.Error, got: %T", err)
	}
	if kinds := cErr.Kinds(); len(kinds) == 0 || kinds[0] != "external_authz" {
		t.Errorf("expected error kind external_authz, got: %v", kinds)
	}
}

func TestExternal_Validate_CalloutURL(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured string
		expURL     string
	}{
		{"no url uses the backend origin", "", "/access/v1/evaluation"},
		{"origin only", "https://pdp.example.com", "https://pdp.example.com/access/v1/evaluation"},
		{"origin with a root path", "https://pdp.example.com/", "https://pdp.example.com/access/v1/evaluation"},
		{"an explicit path is kept", "https://pdp.example.com/check", "https://pdp.example.com/check"},
		{"a relative path is kept", "/check", "/check"},
		{"origin with a query", "https://pdp.example.com?tenant=a",
			"https://pdp.example.com/access/v1/evaluation?tenant=a"},
		{"explicit path with a query", "https://pdp.example.com/check?tenant=a",
			"https://pdp.example.com/check?tenant=a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calloutURL string
			external := newTestExternal("test_ac", tc.configured, false, "", roundTripperFunc(
				func(req *http.Request) (*http.Response, error) {
					calloutURL = req.URL.String()
					return respondAllow()(req)
				}))

			req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
			if err := external.Validate(req); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if calloutURL != tc.expURL {
				t.Errorf("expected callout url %q, got: %q", tc.expURL, calloutURL)
			}
		})
	}
}

func TestExternal_Validate_RequestIDCorrelation(t *testing.T) {
	transport, calloutReq, _ := captureCallout(respondAllow())
	external := newTestExternal("test_ac", "http://authz.service/check", false, "", transport)

	req := httptest.NewRequest(http.MethodGet, "http://client.request/protected", nil)
	req = req.WithContext(context.WithValue(req.Context(), request.UID, "couper-uid-1"))

	if err := external.Validate(req); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if id := (*calloutReq).Header.Get("X-Request-ID"); id != "couper-uid-1" {
		t.Errorf("expected the couper request id, got: %q", id)
	}
}
