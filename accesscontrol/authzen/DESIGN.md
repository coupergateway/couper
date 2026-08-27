# AuthZEN Authorization — Design Notes

Requirements for the `authzen` access control (#873), derived from a concrete
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

`authzen` solves both by moving the disjunction into the callout service: a
single access control whose backend decides which credential type it received and
whether it is valid — the same role Envoy's `ext_authz` filter plays.

## Requirements

### 1. Context propagation into `request.context.<label>.*` (prerequisite)

Allow/deny alone is not enough. When `authzen` replaces a `jwt` block (which
it must, to get OR semantics), downstream HCL loses `request.context.<jwt>.claims`.
The callout response (validated claims: subject, granted permissions, organization,
…) must land in the evaluation context, analogous to how the `beta_oauth2` callback
stores its token response in the access-control context map
(`accesscontrol/oauth2.go`). Without this, `authzen` cannot feed
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

- callout `401` → `authzen_invalid_credentials` (super type `access_control`),
  default 401 — handlers need this to emit RFC 6750 challenges, including the
  RFC 9728 `WWW-Authenticate: Bearer resource_metadata="..."` pointer that MCP
  clients use for discovery.
- callout `403` → deny with `insufficient_permissions` semantics, default 403.

The 401/403 distinction is load-bearing for OAuth resources: `invalid_token` tells
the client to (re)acquire a token; `insufficient_scope` tells it not to bother.

### 4. Callout latency — persistent HTTP/2, no gateway-side decision caching

MCP (and JSON-RPC in general) funnels every operation through one `POST` endpoint, so
a synchronous callout per request doubles request latency on the hottest path.

Decided against an opt-in decision cache in the gateway: a decision is a function of
whatever the service looked at (credential, path, method, TLS state), which the gateway
cannot know — the same reason Envoy's `ext_authz` never shipped result caching. The
authorization service caches internally where its decision allows it.

Instead the callout cost is reduced Envoy-style via connection reuse: a `backend` with
`http2 = true` multiplexes all callouts over one persistent HTTP/2 (TLS/ALPN) connection
to the — typically local — authorization service; without it, HTTP/1.1 keep-alive still
avoids per-request connection setup. For a trusted cleartext origin `http2_prior_knowledge`
gives the same multiplexing without TLS (h2c).

### 5. Keep request-body forwarding out of the initial scope

Body-level decisions (e.g. per-tool filtering of JSON-RPC calls) are owned by
`beta_mcp_proxy` (#935), which already parses the protocol. Keeping the callout
context header/route-only keeps it small and cacheable; body forwarding can be added
later behind an explicit opt-in with a size cap.

### 6. Wire format: OpenID AuthZEN Authorization API 1.0

Resolved. The callout speaks the AuthZEN Authorization API 1.0 (Final, 2026-01-11): Couper is
the policy enforcement point, the authorization service is the policy decision point. This
replaces the Couper-only context schema this document first sketched, and it makes existing
authorizers (Topaz/Aserto, Axiomatics, SGNL, Cerbos, OPA) usable without adapters. Couper joins
the interop peer group of Envoy, Kong, Tyk, Zuplo and the AWS API Gateway. Envoy's
`CheckRequest`/`CheckResponse` is no longer worth a second encoder.

Two consequences deserve a note, because they invert what §3 assumed:

- **A deny is in-band.** AuthZEN answers `200` with `{"decision": false}`. An error status of
  the decision point is a fault between it and Couper, not a statement about the client — a
  `401` says that *Couper* failed to authenticate to the decision point. Couper therefore
  copies nothing from a non-`200` response; forwarding that challenge would mislead the client.
- **The 401/403 split needs a convention.** The spec leaves the response `context` free-form.
  Couper reads one property of it, `www_authenticate`, and answers `401` with that challenge
  when it is present. Every other deny is a `403`. A decision point that sends only
  `{"decision": false}` therefore works without Couper-specific configuration. This convention
  is Couper's own and the documentation says so.

### 7. Authentication belongs to the decision point

AuthZEN assumes that the enforcement point already resolved a subject; requirement §1 assumes
the opposite, because the whole point is to let the callout decide which credential type it
received. Couper resolves this in favour of the decision point: it validates nothing and sends
the raw bearer token as `subject.id` with the type `JWT` — the fallback the AuthZEN gateway
profile defines for exactly this case. A request without a bearer token is `anonymous`; its
credential travels in `context.headers`, where the decision point reads it. A deployment that
does authenticate at the gateway configures `subject` explicitly and gets the interop-standard
`identity`/`sub` shape.

## Client-flow note

The callout is invisible to clients — there is no redirect. With `client_credentials`
the client obtains its token directly from the authorization server (discovered via
the 401 challenge → protected-resource metadata → RFC 8414 AS metadata → token
endpoint) and simply presents the bearer to the gateway; `authzen` then runs
as a synchronous sidecar check. The same holds for `authorization_code` — the flow a
client uses to obtain the credential is orthogonal to how the gateway authorizes it.
