# Response

The `response` block creates and sends a client response.

| Block name | Context                                         | Label    |
|:-----------|:------------------------------------------------|:---------|
| `response` | [Endpoint Block](/configuration/block/endpoint) | no label |

The response body can be omitted or must be one of `body` or `json_body`.

::attributes
---
values: [
  {
    "default": "",
    "description": "Response body which creates implicit default `Content-Type: text/plain` header field.",
    "name": "body",
    "type": "string"
  },
  {
    "default": "",
    "description": "Same as `set_response_headers` in [Modifiers - Response Header](../modifiers#response-header).",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "JSON response body which creates implicit default `Content-Type: application/json` header field.",
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple"
  },
  {
    "default": "200",
    "description": "The HTTP status code to return.",
    "name": "status",
    "type": "number"
  }
]

---
::
