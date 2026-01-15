# Proxy

The `proxy` block creates and executes a proxy request to a backend service.

> 📝 Multiple `proxy` and [`request`](/configuration/block/request) blocks are executed in parallel.

| Block name | Context                                         | Label                          |
|:-----------|:------------------------------------------------|:-------------------------------|
| `proxy`    | [Endpoint Block](/configuration/block/endpoint) | See `Label` description below. |

**Label:** If defined in an [Endpoint Block](/configuration/block/endpoint), a `proxy` block or [Request Block](/configuration/block/request) w/o a label has an implicit name `"default"`. If defined in the [Definitions Block](/configuration/block/definitions), the label of `proxy` is used as reference in [Endpoint Blocks](/configuration/block/endpoint) and the name can be defined via `name` attribute. Only **one** `proxy` block or [Request Block](/configuration/block/request) w/ label `"default"` per [Endpoint Block](/configuration/block/endpoint) is allowed. 

::attributes
---
values: [
  {
    "default": "",
    "description": "Key/value pairs to add form parameters to the upstream request body.",
    "name": "add_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add query parameters to the upstream request URL.",
    "name": "add_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as request headers in the upstream request.",
    "name": "add_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as response headers in the client response.",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the proxy request. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code is not included in this list an `unexpected_status` error will be thrown which can be handled with an [`error_handler`](error_handler). Mutually exclusive with `unexpected_status`.",
    "name": "expected_status",
    "type": "tuple (int)"
  },
  {
    "default": "\"default\"",
    "description": "Defines the proxy request name. Allowed only in the [`definitions` block](definitions).",
    "name": "name",
    "type": "string"
  },
  {
    "default": "",
    "description": "List of names to remove form parameters from the upstream request body.",
    "name": "remove_form_params",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "List of names to remove query parameters from the upstream request URL.",
    "name": "remove_query_params",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the upstream request.",
    "name": "remove_request_headers",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the client response.",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as request headers in the upstream request.",
    "name": "set_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as response headers in the client response.",
    "name": "set_response_headers",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code is included in this list an `unexpected_status` error will be thrown which can be handled with an [`error_handler`](error_handler). Mutually exclusive with `expected_status`.",
    "name": "unexpected_status",
    "type": "tuple (int)"
  },
  {
    "default": "",
    "description": "URL of the resource to request. May be relative to an origin specified in a referenced or nested `backend` block.",
    "name": "url",
    "type": "string"
  },
  {
    "default": "false",
    "description": "Allows support for WebSockets. This attribute is only allowed in the \"default\" proxy block. Other `proxy` blocks, `request` blocks or `response` blocks are not allowed within the current `endpoint` block. Mutually exclusive with `websockets` block.",
    "name": "websockets",
    "type": "bool"
  }
]

---
::

If the `url` attribute is specified and its value is an absolute URL, the protocol and host parts must be the same as in the value of the {origin} attribute of the used backend.

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for the proxy request (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures support for [websockets](/configuration/block/websockets) connections (zero or one). Mutually exclusive with `websockets` attribute.",
    "name": "websockets"
  }
]

---
::
