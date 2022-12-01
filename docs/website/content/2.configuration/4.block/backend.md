---
title: 'Backend'
description: 'The backend defines the connection pool with given origin for outgoing connections.'
draft: false
---

# Backend

The `backend` block defines the connection to a local/remote backend service.

Backends can be defined in the [Definitions Block](/configuration/block/definitions) and referenced by _label_.

| Block name | Context                                                                                                                                                                                                                                   | Label                                                                     | Nested block(s)                                                                                                                       |
|:-----------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------|
| `backend`  | [Definitions Block](/configuration/block/definitions), [Proxy Block](/configuration/block/proxy), [Request Block](/configuration/block/request), [OAuth2 CC Block](/configuration/block/oauth2req_auth), [JWT Block](/configuration/block/jwt), [OAuth2 AC (Beta) Block](/configuration/block/beta_oauth2), [OIDC Block](/configuration/block/oidc)                                               | &#9888; required, when defined in [Definitions Block](/configuration/block/definitions)        | [OpenAPI Block](/configuration/block/openapi), [OAuth2 CC Block](/configuration/block/oauth2req_auth), [Health Block](/configuration/block/health), [Token Request (Beta) Block](/configuration/block/token_request), [Rate Limit Block (beta)](/configuration/block/rate_limit), [TLS Block](/configuration/block/backend_tls) |

::attributes
---
values: [
  {
    "default": "",
    "description": "key/value pairs to add form parameters to the upstream request body",
    "name": "add_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add query parameters to the upstream request URL",
    "name": "add_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add as request headers in the upstream request",
    "name": "add_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add as response headers in the client response",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Basic auth for the upstream request with format user:pass .",
    "name": "basic_auth",
    "type": "string"
  },
  {
    "default": "\"10s\"",
    "description": "The total timeout for dialing and connect to the origin.",
    "name": "connect_timeout",
    "type": "duration"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "false",
    "description": "Disables the peer certificate validation. Must not be used in backend refinement.",
    "name": "disable_certificate_validation",
    "type": "bool"
  },
  {
    "default": "false",
    "description": "Disables reusage of connections to the origin. Must not be used in backend refinement.",
    "name": "disable_connection_reuse",
    "type": "bool"
  },
  {
    "default": "",
    "description": "Value of the HTTP host header field for the origin request. Since hostname replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin.",
    "name": "hostname",
    "type": "string"
  },
  {
    "default": "false",
    "description": "Enables the HTTP2 support. Must not be used in backend refinement.",
    "name": "http2",
    "type": "bool"
  },
  {
    "default": "0",
    "description": "The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. Must not be used in backend refinement.",
    "name": "max_connections",
    "type": "number"
  },
  {
    "default": "",
    "description": "URL to connect to for backend requests.",
    "name": "origin",
    "type": "string"
  },
  {
    "default": "",
    "description": "Changeable part of upstream URL.",
    "name": "path",
    "type": "string"
  },
  {
    "default": "",
    "description": "Prefixes all backend request paths with the given prefix",
    "name": "path_prefix",
    "type": "string"
  },
  {
    "default": "",
    "description": "A proxy URL for the related origin request.",
    "name": "proxy",
    "type": "string"
  },
  {
    "default": "",
    "description": "list of names to remove form parameters from the upstream request body",
    "name": "remove_form_params",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "list of names to remove query parameters from the upstream request URL",
    "name": "remove_query_params",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "list of names to remove headers from the upstream request",
    "name": "remove_request_headers",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "list of names to remove headers from the client response",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL",
    "name": "set_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL",
    "name": "set_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to set as request headers in the upstream request",
    "name": "set_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to set as response headers in the client response",
    "name": "set_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Modifies the response status code.",
    "name": "set_response_status",
    "type": "number"
  },
  {
    "default": "\"300s\"",
    "description": "The total deadline duration a backend request has for write and read/pipe.",
    "name": "timeout",
    "type": "duration"
  },
  {
    "default": "\"60s\"",
    "description": "The duration from writing the full request to the origin and receiving the answer.",
    "name": "ttfb_timeout",
    "type": "duration"
  },
  {
    "default": "false",
    "description": "Ignores the health state and continues with the outgoing request",
    "name": "use_when_unhealthy",
    "type": "bool"
  }
]

---
::

::duration
---
---
::

::blocks
---
values: [
  {
    "description": "Configures a [health check](/configuration/block/health).",
    "name": "beta_health"
  },
  {
    "description": "Configures [rate limiting](/configuration/block/rate_limit).",
    "name": "beta_rate_limit"
  },
  {
    "description": "Configures [OpenAPI validation](/configuration/block/openapi).",
    "name": "openapi"
  },
  {
    "description": "Configures [backend TLS](/configuration/block/backend_tls).",
    "name": "tls"
  }
]

---
::

## Refining a referenced backend

Referenced backends may be "refined" by using a labeled `backend` block in places where an unlabeled `backend` block would also be allowed, e.g. in a `proxy` block:

```hcl
    proxy {
      backend "ref_be" {      # refine referenced backend
        path = "/b"           # override existing attribute value
        add_form_params = {   # set new attribute
          # ...
        }
      }
    }

# ...

definitions {
  backend "ref_be" {
    origin = "https://example.com"
    path = "/a"
  }
}
```

If an attribute is set in both the _referenced_ and the _refining_ block, the value in the _refining_ block is used.

**Note:** Child _blocks_ and the following _attributes_ are not allowed in refining `backend` blocks:
* `disable_certificate_validation`,
* `disable_connection_reuse`,
* `http2` and
* `max_connections`.
