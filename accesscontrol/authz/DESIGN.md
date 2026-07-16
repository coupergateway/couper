# External Authorization — Design Notes

Requirements for the `authz_external` access control (#873), derived from a concrete
use case: fronting a remote MCP server (streamable HTTP) that acts as an OAuth 2.0
protected resource (RFC 9728), where the authorization server issues bearer tokens via
the `client_credentials` grant and clients discover it through the
`WWW-Authenticate: Bearer resource_metadata="..."` challenge.

## Why the existing access controls don't cover this

The gateway must accept two credential types on the same route: an OAuth 2.0 bearer
JWT **or** a static/opaque API key sent in a request header. This is currently not
expressible:

- The `access_control` list is conjunctive — all listed controls must pass. There is
  no native "either A or B" combination, and `error_handler` cannot delegate to a
  second access control.
- There is no API-key access control; opaque (non-JWT) keys cannot be validated by
  any existing block.

`authz_external` solves both by moving the disjunction into the callout service: a
single access control whose backend decides which credential type it received and
whether it is valid — the same role Envoy's `ext_authz` filter plays.

## Requirements

### 1. Context propagation into `request.context.<label>.*` (prerequisite)

Allow/deny alone is not enough. When `authz_external` replaces a `jwt` block (which
it must, to get OR semantics), downstream HCL loses `request.context.<jwt>.claims`.
The callout response (validated claims: subject, granted permissions, organization,
…) must land in the evaluation context, analogous to how the `beta_oauth2` callback
stores its token response in the access-control context map
(`accesscontrol/oauth2.go`). Without this, `authz_external` cannot feed
claim-driven features such as `required_permission`, `permissions_claim`-style
mapping, or `beta_mcp_proxy`'s runtime-evaluated `allowed_tools` (#935).

### 2. Upstream header mutation from the callout response

Parity with Envoy `ext_authz` `allowed_upstream_headers`: the authz service should be
able to return headers that the gateway sets on the proxied request (e.g. resolved
identity, or a re-signed internal token via `jwt_sign()`), so backends behind the
gateway can trust a single internal issuer instead of re-validating every credential
type themselves.

### 3. Distinct error types + `error_handler` support

Register error types for the new access control (compare `beta_mcp_tool_blocked` in
#935) and map the callout's response status instead of collapsing everything into
"non-200 = deny":

- callout `401` → `authz_external_invalid_credentials` (super type `access_control`),
  default 401 — handlers need this to emit RFC 6750 challenges, including the
  RFC 9728 `WWW-Authenticate: Bearer resource_metadata="..."` pointer that MCP
  clients use for discovery.
- callout `403` → deny with `insufficient_permissions` semantics, default 403.

The 401/403 distinction is load-bearing for OAuth resources: `invalid_token` tells
the client to (re)acquire a token; `insufficient_scope` tells it not to bother.

### 4. Opt-in decision caching

MCP (and JSON-RPC in general) funnels every operation through one `POST` endpoint, so
a synchronous callout per request doubles request latency on the hottest path. Offer
an opt-in TTL cache in the existing `cache.MemoryStore` keyed on a hash of the
presented credential (never the raw value), analogous to the backend `oauth2` token
cache in `handler/transport/oauth2_req_auth.go`.

### 5. Keep request-body forwarding out of the initial scope

Body-level decisions (e.g. per-tool filtering of JSON-RPC calls) are owned by
`beta_mcp_proxy` (#935), which already parses the protocol. Keeping the callout
context header/route-only keeps it small and cacheable; body forwarding can be added
later behind an explicit opt-in with a size cap.

### 6. Consider Envoy `ext_authz` wire compatibility

HTTP and gRPC callouts are both planned. Implementing (or optionally supporting)
Envoy's `CheckRequest`/`CheckResponse` contract would make existing authorizers
(OpenFGA, OPA/plugins, oathkeeper-style services) usable without adapters, instead of
introducing a Couper-only context schema.

## Client-flow note

The callout is invisible to clients — there is no redirect. With `client_credentials`
the client obtains its token directly from the authorization server (discovered via
the 401 challenge → protected-resource metadata → RFC 8414 AS metadata → token
endpoint) and simply presents the bearer to the gateway; `authz_external` then runs
as a synchronous sidecar check. The same holds for `authorization_code` — the flow a
client uses to obtain the credential is orthogonal to how the gateway authorizes it.
