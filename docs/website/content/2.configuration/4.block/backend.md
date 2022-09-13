---
title: 'Backend'
description: 'The backend defines the connection pool with given origin for outgoing connections.'
draft: false
---

# Backend

The `backend` block defines the connection to a local/remote backend service.

Backends can be defined in the [Definitions Block](definitions) and referenced by _label_.

| Block name | Context                                                                                                                                                                                                                                   | Label                                                                     | Nested block(s)                                                                                                                       |
|:-----------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------|
| `backend`  | [Definitions Block](definitions), [Proxy Block](proxy), [Request Block](request), [OAuth2 CC Block](oauth2req_auth), [JWT Block](jwt), [OAuth2 AC Block (beta)](oauth2), [OIDC Block](oidc)                                               | &#9888; required, when defined in [Definitions Block](definitions)        | [OpenAPI Block](openapi), [OAuth2 CC Block](oauth2req_auth), [Health Block](health), [Token Request (Beta) Block](token_request), [Rate Limit Block (beta)](rate_limit) |

::attributes
---
values: [
  {
    "name": "add_form_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add form parameters to the upstream request body"
  },
  {
    "name": "add_query_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add query parameters to the upstream request URL"
  },
  {
    "name": "add_request_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as request headers in the upstream request"
  },
  {
    "name": "add_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as response headers in the client response"
  },
  {
    "name": "basic_auth",
    "type": "string",
    "default": "",
    "description": "Basic auth for the upstream request with format user:pass ."
  },
  {
    "name": "connect_timeout",
    "type": "duration",
    "default": "\"10s\"",
    "description": "The total timeout for dialing and connect to the origin."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "disable_certificate_validation",
    "type": "bool",
    "default": "false",
    "description": "Disables the peer certificate validation. Must not be used in backend refinement."
  },
  {
    "name": "disable_connection_reuse",
    "type": "bool",
    "default": "false",
    "description": "Disables reusage of connections to the origin. Must not be used in backend refinement."
  },
  {
    "name": "hostname",
    "type": "string",
    "default": "",
    "description": "Value of the HTTP host header field for the origin request. Since hostname replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin."
  },
  {
    "name": "http2",
    "type": "bool",
    "default": "false",
    "description": "Enables the HTTP2 support. Must not be used in backend refinement."
  },
  {
    "name": "max_connections",
    "type": "number",
    "default": "0",
    "description": "The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. Must not be used in backend refinement."
  },
  {
    "name": "origin",
    "type": "string",
    "default": "",
    "description": "URL to connect to for backend requests."
  },
  {
    "name": "path",
    "type": "string",
    "default": "",
    "description": "Changeable part of upstream URL."
  },
  {
    "name": "path_prefix",
    "type": "string",
    "default": "",
    "description": "Prefixes all backend request paths with the given prefix"
  },
  {
    "name": "proxy",
    "type": "string",
    "default": "",
    "description": "A proxy URL for the related origin request."
  },
  {
    "name": "remove_form_params",
    "type": "object",
    "default": "",
    "description": "list of names to remove form parameters from the upstream request body"
  },
  {
    "name": "remove_query_params",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove query parameters from the upstream request URL"
  },
  {
    "name": "remove_request_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the upstream request"
  },
  {
    "name": "remove_response_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the client response"
  },
  {
    "name": "set_form_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL"
  },
  {
    "name": "set_query_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL"
  },
  {
    "name": "set_request_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set as request headers in the upstream request"
  },
  {
    "name": "set_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set as response headers in the client response"
  },
  {
    "name": "set_response_status",
    "type": "number",
    "default": "",
    "description": "Modifies the response status code."
  },
  {
    "name": "timeout",
    "type": "duration",
    "default": "\"300s\"",
    "description": "The total deadline duration a backend request has for write and read/pipe."
  },
  {
    "name": "ttfb_timeout",
    "type": "duration",
    "default": "\"60s\"",
    "description": "The duration from writing the full request to the origin and receiving the answer."
  },
  {
    "name": "use_when_unhealthy",
    "type": "bool",
    "default": "false",
    "description": "Ignores the health state and continues with the outgoing request"
  }
]

---
::

::duration
---
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
