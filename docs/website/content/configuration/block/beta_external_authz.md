---
title: 'External Authorization (Beta)'
slug: 'beta_external_authz'
description: 'The beta_external_authz block lets you delegate the authorization decision for client requests to an external service.'
---

# External Authorization (Beta)

| Block name            | Context                                                | Label            |
|:----------------------|:-------------------------------------------------------|:-----------------|
| `beta_external_authz` | [Definitions Block](/configuration/block/definitions)  | &#9888; required |

The `beta_external_authz` block lets you delegate the authorization decision for client
requests to an external service. Like all [access control](/configuration/access-control)
types, the `beta_external_authz` block is defined in the
[`definitions` block](/configuration/block/definitions) and can be referenced in all
configuration blocks by its required _label_.

For every protected request Couper sends a `POST` request to the configured authorization
service. The body is an access evaluation request of the
[OpenID AuthZEN Authorization API 1.0](https://openid.net/specs/authorization-api-1_0.html).
Couper is the policy enforcement point (PEP), the authorization service is the policy decision
point (PDP):

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

The `subject` names the credential, not a validated principal. Couper does not validate the
credential, the authorization service does. For a request with a bearer token the type is `JWT`
and the `id` is the raw token. For a request without a bearer token the type is `anonymous`. Its
credential, an API key for example, is still in `context.headers`, where the authorization
service reads it.

The `resource` names the matched route, because a policy applies to the route and not to a
single request path. The `id` keeps the placeholders of the route, for example `/todos/{todoId}`.
If no route matched, for example in front of a [`files` block](/configuration/block/files), the
type is `uri` and the `id` is the request path.

`context.headers` holds all request headers with lower-case names and the first value of each
header, like the [`request.headers` variable](/configuration/variables#request).

## Shaping the evaluation request

The `subject`, `action`, `resource` and `context` attributes shape the callout. Couper
evaluates them for every request, so an access control in front of `beta_external_authz` can
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

  beta_external_authz "authz" {
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

`subject`, `action` and `resource` **replace** their default, because each is a closed record
with mandatory members and a partial merge would make a confusing hybrid. A `subject` or a
`resource` needs a `type` and an `id`, an `action` needs a `name`; an empty value denies the
request. An optional `properties` object is passed through.

`context` **merges over** the defaults, because it is an open bag and `headers` and `tls` are
additive. A configured key wins over a default of the same name.

With `include_tls = true` Couper adds the TLS connection state of the client request to
`context.tls`. This state is a fact about the request, not a statement about the principal: the
certificate can belong to a mesh sidecar while a bearer token identifies the caller. In a
client-facing mTLS setup the `client_certificate` carries the fields an authorization service
keys on — this is the full object such a service can expect:

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

`serial_number` is hex-encoded and `fingerprint_sha256` is the hex SHA-256 of the DER
certificate — use either for allow lists or pinning. The subject alternative names
(`dns_names`, `uris`, `email_addresses`, `ip_addresses`) appear only when the certificate
carries them and often hold the identity to authorize on, e.g. a SPIFFE ID in `uris`.

## Reaching the authorization service

The default path is the AuthZEN access evaluation endpoint `/access/v1/evaluation`, so an
origin is enough to reach a conformant service. A `url` with a path of its own is used as
configured. Couper sends its request id as `X-Request-ID`, which a decision point echoes to
tie its log to the [Couper log](/observation/logging).

With `configuration_url` Couper reads the endpoint from the AuthZEN configuration document of
the authorization service instead. `configuration_url` and `url` are mutually exclusive.

```hcl
definitions {
  beta_external_authz "authz" {
    configuration_url = "https://pdp.example.com/.well-known/authzen-configuration"
    configuration_ttl = "10m"
  }
}
```

Couper takes `access_evaluation_endpoint` from the document, or
`access_evaluations_endpoint` with `evaluate_permissions`. It caches the document for
`configuration_ttl` and keeps a stale copy for `configuration_max_stale` while the service is
unreachable.

Two checks protect the callout. The document must claim the `policy_decision_point` Couper
asked, which is `configuration_url` without the well-known suffix — this is how the
specification prevents a mix-up between decision points. And the endpoint must stay on the
origin of the document: a `backend` pins scheme and host, so an endpoint on a foreign origin
would send the credentials of the client to the configured host instead. Couper denies the
request in both cases.

Couper calls the authorization service on the hot path of every protected request, so the
connection to it should be persistent. This is the recommended setup: a (typically local)
authorization service behind a `backend` with `http2 = true` — callouts are then multiplexed
over a single persistent HTTP/2 connection instead of paying a round trip per request.
HTTP/2 is negotiated via TLS (ALPN), so `http2` applies to `https` origins. For a trusted
cleartext (`http`) authorization service, `http2_prior_knowledge = true` establishes the
same multiplexed HTTP/2 connection without TLS (h2c) — the service must speak HTTP/2.
Without either, Couper still reuses connections (HTTP/1.1 keep-alive), just without
multiplexing.

```hcl
definitions {
  beta_external_authz "authz" {
    backend {
      origin = "https://localhost:4000" # callout to /access/v1/evaluation
      http2  = true
    }
  }
}
```

Couper does not cache authorization decisions: whether a decision may be reused is only
known to the authorization service, which can cache internally whenever its decision
allows it.

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
| `200`, `"decision": false` with a `www_authenticate` challenge | Denied with error type `external_authz_invalid_credentials`, default response status `401`. |
| `200`, `"decision": false`                       | Denied with error type `external_authz_insufficient_permissions`, default response status `403`.   |
| `200` without a `decision`, or a malformed body  | Denied with error type `external_authz`, default response status `403`.                            |
| any other status, or a callout failure           | Denied with error type `external_authz`, default response status `403`.                            |

An error status of the authorization service reports a problem between Couper and that
service, not a denied client. A `401`, for example, says that Couper failed to authenticate
to the authorization service. Couper copies nothing from such a response, because its
challenge is addressed to Couper and would mislead the client. An
[`error_handler` block](/configuration/error-handling) for the `external_authz` type catches
every such rejection — a `400` for an action unknown to the decision point's model, for
example, stays a plain denial with a body of your choosing.

The response `context` is exposed as the
[`request.context.<label>` variable](/configuration/variables#context) — the place for
validated claims, the resolved identity or granted permissions. Couper adds two members:
`decision`, and `headers` with the callout response headers (lower-cased names, first value,
like `request.headers`). Both shadow a response context property of the same name. Couper
exposes the `context` of a denial as well, so an `error_handler` can read the reason.

An upstream backend can trust a resolved identity or a re-signed internal token (created with
[`jwt_sign()`](/configuration/functions)) the authorization service returns as a header, by
copying it onto the request with `set_request_headers` — which overwrites any client-provided
value:

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


With `permissions_property` the authorization service can grant [permissions](/configuration/error-handling#permissions-related-error_handler)
evaluated by `required_permission` in [`api`](/configuration/block/api) or [`endpoint`](/configuration/block/endpoint)
blocks: the named property of the response `context` — a space-separated string or a list of strings, like the
[`jwt` block's](/configuration/block/jwt) `permissions_claim` — is added to `request.context.granted_permissions`.
If the configured property is absent from an allowed response, the request is denied rather than allowed
without permissions, matching the fail-closed handling of a malformed body.

```hcl
definitions {
  beta_external_authz "authz" {
    url                  = "https://authz.example.com/check"
    permissions_property = "granted_permissions"
  }
}
```

`permissions_property` needs an authorization service that hands out a permission set. Most
policy engines do not; they answer one question at a time. `evaluate_permissions` asks them in
their own terms: Couper names candidate permissions, and one batch callout to the AuthZEN
access evaluations endpoint `/access/v1/evaluations` asks about the client request and about
every candidate. Couper grants the permissions the service allows.

```hcl
definitions {
  beta_external_authz "authz" {
    backend              = "pdp" # callout to /access/v1/evaluations
    evaluate_permissions = ["can_read", "can_write", "can_delete"]
  }
}
```

The first entry of the `evaluations` array is the client request, the others follow the order
of `evaluate_permissions`. Couper sends `options.evaluations_semantic = "execute_all"`, because
a short-circuit semantic truncates the answers and would lose permissions. The first decision
allows or denies the request; a response with fewer answers than questions denies it.

A `url` with a path of its own is used as configured — with `evaluate_permissions` it must
point to the access evaluations endpoint, not to the single-evaluation endpoint.

A [`required_permission`](/configuration/block/endpoint) of the protected `endpoint` or `api`
block replaces the candidates for that request: Couper resolves it at callout time — the
expression sees everything a preceding access control provided — and asks about exactly the
permission the endpoint will check. The permission declaration stays at the endpoint, and the
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
  beta_external_authz "authz" {
    backend              = "pdp" # callout to /access/v1/evaluations
    evaluate_permissions = ["can_read", "can_write", "can_delete"]
  }
}
```

Endpoints without a required permission — and requests whose method has no entry in a
required-permission map — keep the configured candidates.

`evaluate_permissions` and `permissions_property` are mutually exclusive.

AuthZEN denies a request with a flat `"decision": false` and leaves the response `context`
free-form. To keep an OAuth 2.0 protected resource workable, Couper reads one property of that
context by convention — this convention is Couper's, not a part of the specification:

```json
{
  "decision": false,
  "context": {
    "www_authenticate": "Bearer resource_metadata=\"https://couper.example.com/.well-known/oauth-protected-resource\""
  }
}
```

A challenge in `context.www_authenticate` means invalid credentials: it tells the client how to
authenticate, for example with an RFC 9728 `resource_metadata` pointer. Couper then answers
`401` and a default `error_handler` forwards the challenge to the client. Without a challenge
the denial is an authorization decision and Couper answers `403`, because new credentials would
not help the client. An authorization service that sends only `{"decision": false}` therefore
works without any Couper-specific configuration.

The challenge is available to custom handlers as
`request.context.<label>.www_authenticate`. Defining an
[`error_handler` block](/configuration/error-handling) for
`external_authz_invalid_credentials` replaces the default:

```hcl
definitions {
  beta_external_authz "authz" {
    url = "https://authz.example.com/check"

    error_handler "external_authz_invalid_credentials" {
      set_response_headers = {
        www-authenticate = "Bearer resource_metadata=\"https://couper.example.com/.well-known/oauth-protected-resource\""
      }
    }
  }
}
```

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
    "description": "Candidate permissions to resolve with one batch callout to the AuthZEN access evaluations endpoint. Couper asks the authorization service about the client request and about every listed permission, and grants those it allows. A `required_permission` of the protected endpoint or API replaces the candidates for that request. Mutually exclusive with `permissions_property`.",
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
    "description": "Name of the property in the response `context` containing the granted permissions. The property value must either be a string containing a space-separated list of permissions or a list of string permissions.",
    "name": "permissions_property",
    "type": "string"
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
