---
title: 'External Authorization (Beta)'
slug: 'beta_authz_external'
description: 'The beta_authz_external block lets you delegate the authorization decision for client requests to an external service.'
---

# External Authorization (Beta)

| Block name            | Context                                                | Label            |
|:----------------------|:-------------------------------------------------------|:-----------------|
| `beta_authz_external` | [Definitions Block](/configuration/block/definitions)  | &#9888; required |

The `beta_authz_external` block lets you delegate the authorization decision for client
requests to an external service. Like all [access control](/configuration/access-control)
types, the `beta_authz_external` block is defined in the
[`definitions` block](/configuration/block/definitions) and can be referenced in all
configuration blocks by its required _label_.

For every protected request Couper sends a `POST` request with a JSON body describing the
client request to the configured authorization service:

```json
{
  "client_request": {
    "method": "GET",
    "url": "https://couper.example.com/protected",
    "headers": {
      "Authorization": ["Bearer ..."]
    }
  }
}
```

With `include_tls = true` the TLS connection information of the client request is added:

```json
{
  "client_request": { "...": "..." },
  "metadata_tls": {
    "version": "TLS 1.3",
    "cipher_suite": "TLS_AES_128_GCM_SHA256",
    "server_name": "couper.example.com",
    "client_certificate": {
      "subject": "CN=my-client",
      "issuer": "CN=my-ca",
      "not_before": "2026-01-01T00:00:00Z",
      "not_after": "2027-01-01T00:00:00Z"
    }
  }
}
```

Couper calls the authorization service on the hot path of every protected request, so the
connection to it should be persistent. This is the recommended setup: a (typically local)
authorization service behind a `backend` with `http2 = true` — callouts are then multiplexed
over a single persistent HTTP/2 connection instead of paying a round trip per request.
HTTP/2 is negotiated via TLS (ALPN), so the authorization service must be reachable via
`https` — without `http2` Couper still reuses connections (HTTP/1.1 keep-alive), just
without multiplexing.

```hcl
definitions {
  beta_authz_external "authz" {
    backend {
      origin = "https://localhost:4000"
      http2  = true
    }
  }
}
```

Couper does not cache authorization decisions: whether a decision may be reused is only
known to the authorization service, which can cache internally whenever its decision
allows it.

The response status code of the authorization service determines the decision:

| Status    | Result                                                                                    |
|:----------|:------------------------------------------------------------------------------------------|
| `200`     | The request is allowed.                                                                     |
| `401`     | Denied with error type `authz_external_invalid_credentials`, default response status `401`. |
| `403`     | Denied with error type `authz_external_insufficient_permissions`, default response status `403`. |
| any other | Denied with error type `authz_external`, default response status `401`.                    |

The `200` response is exposed as the [`request.context.<label>` variable](/configuration/variables#context):
the properties of a JSON object body (`Content-Type: application/json`) — the place for validated
claims, the resolved identity or granted permissions — plus the response headers under
`request.context.<label>.headers` (lower-cased names, first value, like `request.headers`).
A malformed JSON body denies the request, as downstream permission checks may rely on this data.
A body property literally named `headers` is shadowed by the response headers.

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
blocks: the named response body property — a space-separated string or a list of strings, like the
[`jwt` block's](/configuration/block/jwt) `permissions_claim` — is added to `request.context.granted_permissions`.

```hcl
definitions {
  beta_authz_external "authz" {
    url               = "https://authz.example.com/check"
    permissions_property = "granted_permissions"
  }
}
```

The error types can be handled with [`error_handler` blocks](/configuration/error-handling),
for example to send a `WWW-Authenticate` challenge pointing OAuth 2.0 clients to the
protected resource metadata (RFC 9728).

```hcl
definitions {
  beta_authz_external "authz" {
    url = "https://authz.example.com/check"

    error_handler "authz_external_invalid_credentials" {
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
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the authorization callout. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "false",
    "description": "Include TLS connection information of the client request in the authorization request.",
    "name": "include_tls",
    "type": "bool"
  },
  {
    "default": "",
    "description": "Name of the response body property containing the granted permissions. The property value must either be a string containing a space-separated list of permissions or a list of string permissions.",
    "name": "permissions_property",
    "type": "string"
  },
  {
    "default": "",
    "description": "URL of the authorization service. Relative URL references are resolved against the origin of a referenced or nested `backend` block.",
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
