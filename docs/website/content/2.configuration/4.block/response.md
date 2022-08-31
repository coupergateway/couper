# Response

The `response` block creates and sends a client response.

| Block name | Context                           | Label    | Nested block(s) |
|:-----------|:----------------------------------|:---------|:----------------|
| `response` | [Endpoint Block](endpoint) | no label | -               |

The response body can be omitted or must be one of `body` or `json_body`.

::attributes
---
values: [
  {
    "name": "body",
    "type": "string",
    "default": "",
    "description": "Response body which creates implicit default `Content-Type: text/plain` header field."
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "Same as `set_response_headers` in [Request Header](../modifiers#response-header)."
  },
  {
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple",
    "default": "",
    "description": "JSON response body which creates implicit default `Content-Type: application/json` header field."
  },
  {
    "name": "status",
    "type": "number",
    "default": "200",
    "description": "The HTTP status code to return."
  }
]

---
::
