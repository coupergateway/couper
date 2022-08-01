# Proxy

The `proxy` block creates and executes a proxy request to a backend service.

> 📝 Multiple `proxy` and [`request`](request) blocks are executed in parallel.

| Block name | Context                           | Label                                                                                                                                                                                                                                          | Nested block(s)                                                                                                                                                                                                                                |
|:-----------|:----------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `proxy`    | [Endpoint Block](endpoint) | &#9888; A `proxy` block or [Request Block](request) w/o a label has an implicit label `"default"`. Only **one** `proxy` block or [Request Block](request) w/ label `"default"` per [Endpoint Block](endpoint) is allowed. | [Backend Block](backend) (&#9888; required, if no [Backend Block](backend) reference is defined or no `url` attribute is set.), [Websockets Block](websockets) (&#9888; Either websockets attribute or block is allowed.) |


::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "\"\"",
    "description": "backend block reference"
  },
  {
    "name": "websockets",
    "type": "object",
    "default": "",
    "description": "Allows support for websockets. This attribute is only allowed in the 'default' proxy block. Other `proxy` blocks, `request` blocks or `response` blocks are not allowed within the current `endpoint` block."
  },
  {
    "name": "expected_status",
    "type": "tuple (int)",
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code not included in this list an `unexpected_status` error will be thrown which can be handled with an [`error_handler`](error_handler)."
  },
  {
    "name": "url",
    "type": "string",
    "default": "\"\"",
    "description": "If defined, the host part of the URL must be the same as the `origin` attribute of the corresponding backend."
  }
]

---
::
