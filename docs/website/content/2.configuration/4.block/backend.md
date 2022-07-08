---
title: 'Backend'
description: 'The backend defines the connection pool with given origin for outgoing connections.'
draft: false
---

# Backend

The `backend` block defines the connection to a local/remote backend service.

Backends can be defined in the [Definitions Block](/configuration/block/definitions) and referenced by _label_.

| Block name | Context                                                                                                                                                                                                                                   | Label                                                                     | Nested block(s)                                                                                  |
|:-----------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| `backend`  | [Definitions Block](#definitions-block), [Proxy Block](#proxy-block), [Request Block](#request-block), [OAuth2 CC Block](#oauth2-block), [JWT Block](#jwt-block), [OAuth2 AC Block (beta)](#beta-oauth2-block), [OIDC Block](#oidc-block) | &#9888; required, when defined in [Definitions Block](#definitions-block) | [OpenAPI Block](#openapi-block), [OAuth2 CC Block](#oauth2-block), [Health Block](#health-block) |

::attributes
---
values: [
  {
    "name": "disable_certificate_validation",
    "type": "bool",
    "default": "false",
    "description": "Disables the peer certificate validation."
  },
  {
    "name": "disable_connection_reuse",
    "type": "bool",
    "default": "false",
    "description": "Disables reusage of connections to the origin."
  },
  {
    "name": "http2",
    "type": "bool",
    "default": "false",
    "description": "Enables the HTTP2 support."
  },
  {
    "name": "max_connections",
    "type": "number",
    "default": "0",
    "description": "The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin."
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
    "default": "10s",
    "description": "The total timeout for dialing and connect to the origin."
  },
  {
    "name": "hostname",
    "type": "string",
    "default": "",
    "description": "Value of the HTTP host header field for the origin request. Since hostname replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "Defines log fields for custom logging."
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
    "name": "set_response_status",
    "type": "number",
    "default": "",
    "description": "Modifies the response status code."
  },
  {
    "name": "ttfb_timeout",
    "type": "duration",
    "default": "60s",
    "description": "The duration from writing the full request to the origin and receiving the answer."
  },
  {
    "name": "timeout",
    "type": "duration",
    "default": "300s",
    "description": "The total deadline duration a backend request has for write and read/pipe."
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
