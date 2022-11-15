# Proxy

The `proxy` block creates and executes a proxy request to a backend service.

> 📝 Multiple `proxy` and [`request`](/configuration/block/request) blocks are executed in parallel.

| Block name | Context                           | Label                                                                                                                                                                                                                                          | Nested block(s)                                                                                                                                                                                                                                |
|:-----------|:----------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `proxy`    | [Endpoint Block](/configuration/block/endpoint) | See `Label` description below. | [Backend Block](/configuration/block/backend) (&#9888; required, if no [Backend Block](/configuration/block/backend) reference is defined or no `url` attribute is set.), [Websockets Block](/configuration/block/websockets) (&#9888; Either websockets attribute or block is allowed.) |

**Label:** If defined in an [Endpoint Block](/configuration/block/endpoint), a `proxy` block or [Request Block](/configuration/block/request) w/o a label has an implicit name `"default"`. If defined in the [Definitions Block](/configuration/block/definitions), the label of `proxy` is used as reference in [Endpoint Blocks](/configuration/block/endpoint) and the name can be defined via `name` attribute. Only **one** `proxy` block or [Request Block](/configuration/block/request) w/ label `"default"` per [Endpoint Block](/configuration/block/endpoint) is allowed. 

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
    "description": "backend block reference",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code not included in this list an `unexpected_status` error will be thrown which can be handled with an [`error_handler`](error_handler).",
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
    "description": "If defined, the host part of the URL must be the same as the `origin` attribute of the corresponding backend.",
    "name": "url",
    "type": "string"
  },
  {
    "default": "false",
    "description": "Allows support for WebSockets. This attribute is only allowed in the \"default\" proxy block. Other `proxy` blocks, `request` blocks or `response` blocks are not allowed within the current `endpoint` block.",
    "name": "websockets",
    "type": "bool"
  }
]

---
::
