---
title: 'AuthZEN Authorization (Beta)'
slug: 'beta_authzen'
description: 'The beta_authzen block lets you delegate the authorization decision for client requests to an external service.'
---

# AuthZEN Authorization (Beta)

| Block name            | Context                                                | Label            |
|:----------------------|:-------------------------------------------------------|:-----------------|
| `beta_authzen` | [Definitions Block](/configuration/block/definitions)  | &#9888; required |

The `beta_authzen` block delegates the authorization decision for client requests to an
external service. Like all [access control](/configuration/access-control) types, it is
defined in the [`definitions` block](/configuration/block/definitions) and referenced by
its required _label_.

> Specification: [OpenID AuthZEN Authorization API 1.0](https://openid.net/specs/authorization-api-1_0.html)

## Enforcement point, decision point and the tuple

Couper is the policy enforcement point (PEP): it gates the request. The authorization
service is the policy decision point (PDP): it decides. For every protected request Couper
sends a `POST` request to the PDP and enforces the returned decision.

Every question to the PDP is the same tuple: may this **subject** perform this **action**
on this **resource**? An open **context** object carries everything else a decision may
need.

This is the **default** evaluation request. Couper builds it from the client request when
no tuple attribute is set:

```json
{
  "subject": {
    "type": "JWT",
    "id": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
  },
  "action": {
    "name": "GET"
  },
  "resource": {
    "type": "route",
    "id": "/todos/{todoId}",
    "properties": {
      "uri": "https://couper.example.com/todos/42?full=1",
      "scheme": "https",
      "hostname": "couper.example.com",
      "path": "/todos/42",
      "route": "/todos/{todoId}",
      "params": { "todoId": "42" },
      "query": { "full": ["1"] },
      "ip": "10.0.0.7"
    }
  },
  "context": {
    "headers": {
      "authorization": "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
    }
  }
}
```

The `subject` names the credential, not a validated principal. The authorization service
validates it, not Couper. With a bearer token, the type is `JWT` and the `id` is the raw
token. Without one, the type is `anonymous`. Other credentials, an API key for example,
stay in `context.headers`.

The `action` is the HTTP method of the request.

The `resource` names the matched route: a policy applies to the route, not to a single
request path. The `id` keeps the route placeholders, for example `/todos/{todoId}`. Without
a matched route, for example in front of a [`files` block](/configuration/block/files), the
type is `uri` and the `id` is the request path.

`context.headers` holds all request headers with lower-case names and the first value of each
header, like the [`request.headers` variable](/configuration/variables#request).

> Note: AuthZEN standardizes the structure of the tuple, not its values. Each decision
> point has its own vocabulary of types. The
> [`subject`, `action`, `resource` and `context` attributes](#shaping-the-evaluation-request)
> replace the defaults to match it, for example with validated claims of a preceding
> `jwt` access control.

## Shaping the evaluation request

The `subject`, `action`, `resource` and `context` attributes shape the callout. Couper
evaluates them for every request, so an access control in front of `beta_authzen` can
name the subject. Access controls run in the order of the `access_control` list:

```hcl
api {
  endpoint "/todos/{id}" {
    access_control = ["token", "authz"]
    # ...
  }
}

definitions {
  jwt "token" {
    signature_algorithm = "HS256"
    key                 = "..."
  }

  beta_authzen "authz" {
    url = "https://pdp.example.com/access/v1/evaluation"

    subject = {
      type = "identity"
      id   = request.context.token.sub
    }

    action = {
      name = request.method == "GET" ? "can_read" : "can_write"
    }

    context = {
      tenant = request.context.token.org
    }
  }
}
```

`subject`, `action` and `resource` **replace** their default: each is a closed record, and
a partial merge would make a confusing hybrid. A `subject` or a `resource` needs a `type`
and an `id`; an `action` needs a `name`. An empty value denies the request. An optional
`properties` object is passed through.

`context` **merges over** the defaults: it is an open bag, and `headers` and `tls` are
additive. A configured key wins over a default of the same name.

### mTLS

`include_tls = true` adds the TLS connection state of the client request to `context.tls`.
The state describes the connection, not the principal: the certificate can belong to a mesh
sidecar while a bearer token identifies the caller. In a client-facing mTLS setup,
`client_certificate` carries the fields an authorization service keys on. This is the full
object:

```json
{
  "context": {
    "tls": {
      "version": "TLS 1.3",
      "cipher_suite": "TLS_AES_128_GCM_SHA256",
      "server_name": "couper.example.com",
      "client_certificate": {
        "subject": "CN=my-client,O=Example",
        "issuer": "CN=my-ca",
        "serial_number": "1267",
        "fingerprint_sha256": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
        "not_before": "2026-01-01T00:00:00Z",
        "not_after": "2027-01-01T00:00:00Z",
        "dns_names": ["client.example"],
        "uris": ["spiffe://example.org/mcp-client"],
        "email_addresses": ["mcp@example.org"],
        "ip_addresses": ["10.0.0.7"]
      }
    }
  }
}
```

> Tip: `serial_number` is hex-encoded and `fingerprint_sha256` is the hex SHA-256 of the DER
> certificate — use either for allow lists or pinning. The subject alternative names
> (`dns_names`, `uris`, `email_addresses`, `ip_addresses`) appear only when the certificate
> carries them and often hold the identity to authorize on, e.g. a SPIFFE ID in `uris`.

## Reaching the authorization service

The default path is the AuthZEN access evaluation endpoint `/access/v1/evaluation`, so an
origin is enough to reach a conformant service. An explicit path in `url` is used as
configured. Couper sends its request id as `X-Request-ID`; a decision point echoes it to
tie both [logs](/observation/logging) together.

Couper calls the authorization service on the hot path of every protected request, so keep
the connection persistent. Recommended: a (typically local) service behind a `backend` with
`http2 = true` — callouts multiplex over one persistent HTTP/2 connection. HTTP/2 negotiates
via TLS (ALPN), so `http2` applies to `https` origins. For a trusted cleartext (`http`)
service, `http2_prior_knowledge = true` establishes the same connection without TLS (h2c).
Without either, Couper still reuses connections (HTTP/1.1 keep-alive), just without
multiplexing.

```hcl
definitions {
  beta_authzen "authz" {
    backend {
      origin = "https://localhost:4000" # callout to /access/v1/evaluation
      http2  = true
    }
  }
}
```

> Note: Couper does not cache decisions. Only the authorization service knows whether a
> decision may be reused, and it can cache internally.

## Auto discovery

With `configuration_url` Couper reads the endpoint from the AuthZEN configuration document of
the authorization service instead. `configuration_url` and `url` are mutually exclusive.

> Specification: [Policy Decision Point Metadata](https://openid.net/specs/authorization-api-1_0.html)

```hcl
definitions {
  beta_authzen "authz" {
    configuration_url = "https://pdp.example.com/.well-known/authzen-configuration"
    configuration_ttl = "10m"
  }
}
```

Couper takes `access_evaluation_endpoint` from the document, or
`access_evaluations_endpoint` with `evaluate_permissions`. It caches the document for
`configuration_ttl` and keeps a stale copy for `configuration_max_stale` while the service is
unreachable.

Two checks protect the callout; each failure denies the request. The document must claim
the `policy_decision_point` Couper asked — `configuration_url` without the well-known
suffix — which prevents a mix-up between decision points. The endpoint must stay on the
origin of the document: a `backend` pins scheme and host, so a foreign endpoint would
receive the client credentials on the configured host.

> Note: A multi-tenant decision point may root its identifiers under a path of its own —
> OpenFGA claims `<origin>/stores/<store_id>` for
> `/.well-known/authzen-configuration/<store_id>`. Couper accepts that as long as the
> origin and the tenant of `configuration_url` stay pinned.

## The decision

The decision is in the response body, not in the response status code. The authorization
service answers `200` with a `decision` and an optional free-form `context`:

```json
{
  "decision": false,
  "context": {
    "reason": "the subject is a viewer of the resource"
  }
}
```

| Response                                        | Result                                                                                           |
|:------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| `200`, `"decision": true`                        | The request is allowed.                                                                            |
| `200`, `"decision": false` with a `www_authenticate` challenge | Denied with error type `authzen_invalid_credentials`, default response status `401`. |
| `200`, `"decision": false`                       | Denied with error type `authzen_insufficient_permissions`, default response status `403`.   |
| `200` without a `decision`, or a malformed body  | Denied with error type `authzen`, default response status `403`.                            |
| any other status, or a callout failure           | Denied with error type `authzen`, default response status `403`.                            |

An error status reports a problem between Couper and the service, not a denied client. A
`401`, for example, says Couper failed to authenticate to the service. Couper copies
nothing from such a response — its challenge would mislead the client. An
[`error_handler` block](/configuration/error-handling) for the `authzen` type catches
every such rejection: a `400` for an action unknown to the decision point's model, for
example, stays a plain denial with a body of your choosing.

The response `context` is exposed as the
[`request.context.<label>` variable](/configuration/variables#context) — the place for
validated claims, the resolved identity or granted permissions. Couper adds two members:
`decision`, and `headers` with the callout response headers (lower-cased names, first value,
like `request.headers`). Both shadow a response context property of the same name. Couper
exposes the `context` of a denial as well, so an `error_handler` can read the reason.

The authorization service can return a resolved identity or a re-signed internal token
(created with [`jwt_sign()`](/configuration/functions)) as a header. Copy it onto the
upstream request with `set_request_headers`; this overwrites any client-provided value:

```hcl
api {
  endpoint "/**" {
    access_control = ["authz"]

    proxy {
      backend = "protected_api"

      set_request_headers = {
        x-resolved-identity = request.context.authz.headers["x-resolved-identity"]
      }
    }
  }
}
```

### Denials and the challenge

AuthZEN denies a request with a flat `"decision": false` and leaves the response `context`
free-form. To keep an OAuth 2.0 protected resource workable, Couper reads one property of
that context by convention. The convention is Couper's, not part of the specification:

```json
{
  "decision": false,
  "context": {
    "www_authenticate": "Bearer resource_metadata=\"https://couper.example.com/.well-known/oauth-protected-resource\""
  }
}
```

A challenge in `context.www_authenticate` means invalid credentials: it tells the client
how to authenticate, for example with an RFC 9728 `resource_metadata` pointer. Couper then
answers `401` and a default `error_handler` forwards the challenge to the client. Without a
challenge the denial is an authorization decision and Couper answers `403` — new credentials
would not help. A service that sends only `{"decision": false}` works without
Couper-specific configuration.

The challenge is available to custom handlers as
`request.context.<label>.www_authenticate`. An
[`error_handler` block](/configuration/error-handling) for
`authzen_invalid_credentials` replaces the default:

```hcl
definitions {
  beta_authzen "authz" {
    url = "https://authz.example.com/check"

    error_handler "authzen_invalid_credentials" {
      set_response_headers = {
        www-authenticate = "Bearer resource_metadata=\"https://couper.example.com/.well-known/oauth-protected-resource\""
      }
    }
  }
}
```

## Batch decisions and permissions

The authorization service can grant [permissions](/configuration/error-handling#permissions-related-error_handler)
evaluated by `required_permission` in [`api`](/configuration/block/api) or [`endpoint`](/configuration/block/endpoint)
blocks. Policy engines answer one question at a time, and `evaluate_permissions` asks them
in their own terms: one batch callout to `/access/v1/evaluations` asks about the client
request and about every candidate permission. Couper grants those the service allows.

> Specification: [Access Evaluations API](https://openid.net/specs/authorization-api-1_0.html)

```hcl
definitions {
  beta_authzen "authz" {
    backend              = "pdp" # callout to /access/v1/evaluations
    evaluate_permissions = ["can_read", "can_write", "can_delete"]
  }
}
```

The first entry of the `evaluations` array is the client request, the others follow the order
of `evaluate_permissions`. Couper sends `options.evaluations_semantic = "execute_all"`, because
a short-circuit semantic truncates the answers and would lose permissions. A response with
fewer answers than questions denies the request.

> Note: A `url` with a path of its own is used as configured — with `evaluate_permissions` it
> must point to the access evaluations endpoint, not to the single-evaluation endpoint.

A [`required_permission`](/configuration/block/endpoint) of the protected `endpoint` or
`api` block replaces the candidates for that request. Couper resolves it at callout time —
the expression sees everything a preceding access control provided — and asks about exactly
the permission the endpoint will check. The declaration stays at the endpoint, and the
batch stays small:

```hcl
api {
  endpoint "/todos/{id}" {
    access_control      = ["authz"]
    required_permission = { GET = "can_read", "*" = "can_write" }
    # a GET asks about the request and about can_read; nothing else
    # ...
  }
}

definitions {
  beta_authzen "authz" {
    backend              = "pdp" # callout to /access/v1/evaluations
    evaluate_permissions = ["can_read", "can_write", "can_delete"]
  }
}
```

Endpoints without a required permission — and requests whose method has no entry in a
required-permission map — keep the configured candidates.

### What a single `false` means

Only the first decision of the batch allows or denies the request. Every further decision
answers a permission question: `true` grants the candidate, `false` only withholds it — it
does not deny the request by itself.

- With the configured candidates, a request passes as long as the first decision is `true`.
  A withheld permission surfaces only where a `required_permission` demands it: as a `403`
  with error type `insufficient_permissions`, raised by Couper's permission check, not by
  the access control.
- With a `required_permission` override the batch carries exactly two questions. The first
  still gates the request; a `false` on the second withholds the one permission the endpoint
  checks, so the request ends in the same `403`.

> Tip: The error type tells the two denials apart in logs and `error_handler` blocks:
> `insufficient_permissions` comes from the permission check, the `authzen_*` types from the
> access control itself.

## Connecting OpenFGA

OpenFGA implements AuthZEN, so it connects like any other decision point — no custom
service required. Its endpoints are scoped to a store: `configuration_url` points at the
store's well-known document. The tuple must speak the model's vocabulary: subject and
resource use the model's types, and the action names a relation on the resource type. With
`evaluate_permissions` the candidates are relations, too.

> Specification: [OpenFGA AuthZEN API](https://openfga.dev/docs/interacting/authzen)

With all three tuple attributes set, one evaluation is exactly an OpenFGA check —
user, relation, object. `GET /documents/42/share` asks whether the caller has `can_share`
on `document:42`:

```hcl
api {
  endpoint "/documents/{id}/{action}" {
    access_control = ["token", "fga"]
    # ...
  }
}

definitions {
  beta_authzen "fga" {
    configuration_url = "https://fga.example.com/.well-known/authzen-configuration/01ARZ3NDEKTSV4RRFFQ69G5FAV"

    subject  = { type = "user", id = request.context.token.sub }   # user
    action   = { name = "can_${request.path_params.action}" }      # relation
    resource = { type = "document", id = request.path_params.id }  # object

    backend {
      set_request_headers = {
        # Uses the model version that CI/CD wrote.
        # Without it, OpenFGA uses the latest model.
        openfga-authorization-model-id = env.FGA_MODEL_ID
      }
    }
  }
}
```

> Note: The AuthZEN API of OpenFGA is experimental and must be enabled with
> `--experimentals=authzen`. Checks that rely on contextual tuples need the native
> OpenFGA API.

## Listing objects

An access control answers one question: may this request pass. Which objects a subject may
see is a data question, and it belongs to the application: the listing is filtered, sorted
and paginated where the data lives. OpenFGA documents three options for this pattern —
search then check, a local index from the changes feed, or list IDs then search — with
guidance on when each fits:

> Specification: [OpenFGA: Search With Permissions](https://openfga.dev/docs/interacting/search-with-permissions)

{{< attributes >}}
[
  {
    "default": "",
    "description": "Replaces the action of the access evaluation request. Requires a `name`; an optional `properties` object is passed through. Defaults to the request method.",
    "name": "action",
    "type": "object"
  },
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the authorization callout. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "\"1h\"",
    "description": "Time after the expiration of the AuthZEN configuration document during which Couper keeps using it. A zero value means no stale use.",
    "name": "configuration_max_stale",
    "type": "duration"
  },
  {
    "default": "\"1h\"",
    "description": "Time to cache the AuthZEN configuration document.",
    "name": "configuration_ttl",
    "type": "duration"
  },
  {
    "default": "",
    "description": "URL of the AuthZEN configuration document (`/.well-known/authzen-configuration`) of the authorization service. Couper reads the callout endpoint from it. Mutually exclusive with `url`.",
    "name": "configuration_url",
    "type": "string"
  },
  {
    "default": "",
    "description": "Merges into the context of the access evaluation request. Configured keys win over the `headers` and `tls` defaults.",
    "name": "context",
    "type": "object"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "Candidate permissions to resolve with one batch callout to the AuthZEN access evaluations endpoint. Couper asks the authorization service about the client request and about every listed permission, and grants those it allows. A `required_permission` of the protected endpoint or API replaces the candidates for that request.",
    "name": "evaluate_permissions",
    "type": "tuple (string)"
  },
  {
    "default": "false",
    "description": "Include TLS connection information of the client request in the authorization request.",
    "name": "include_tls",
    "type": "bool"
  },
  {
    "default": "",
    "description": "Replaces the resource of the access evaluation request. Requires a `type` and an `id`; an optional `properties` object is passed through. Defaults to the matched route.",
    "name": "resource",
    "type": "object"
  },
  {
    "default": "",
    "description": "Replaces the subject of the access evaluation request. Requires a `type` and an `id`; an optional `properties` object is passed through. Defaults to the bearer token of the client request.",
    "name": "subject",
    "type": "object"
  },
  {
    "default": "\"/access/v1/evaluation\"",
    "description": "URL of the authorization service. Relative URL references are resolved against the origin of a referenced or nested `backend` block. Without a path, or with only the root path `/`, the AuthZEN access evaluation endpoint `/access/v1/evaluation` is used — or `/access/v1/evaluations` with `evaluate_permissions`. An explicit path must point to the matching endpoint.",
    "name": "url",
    "type": "string"
  }
]
{{< /attributes >}}

{{< blocks >}}
[
  {
    "description": "Configures a [backend](/configuration/block/backend) for the authorization callout (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]
{{< /blocks >}}
