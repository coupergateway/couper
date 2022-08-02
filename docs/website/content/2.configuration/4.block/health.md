# Health

Defines a recurring health check request for its backend. Results can be obtained via the [`backends.<label>.health` variables](../variables#backends).
Changes in health states and related requests will be logged. Default User-Agent will be `Couper / <version> health-check` if not provided
via `headers` attribute. An unhealthy backend will return with a [`backend_unhealthy`](../error-handling#api-error-types) error.

| Block name    | Context                           | Label | Nested block |
|:--------------|:----------------------------------|:------|:-------------|
| `beta_health` | [`backend` block](backend) | –     |              |

::attributes
---
values: [
  {
    "name": "expected_status",
    "type": "tuple (int)",
    "default": "[200, 204, 301]",
    "description": "one of wanted response status code"
  },
  {
    "name": "expected_text",
    "type": "string",
    "default": "\"\"",
    "description": "text which the response body must contain"
  },
  {
    "name": "failure_threshold",
    "type": "number",
    "default": "2",
    "description": "failed checks needed to consider backend unhealthy"
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "request headers"
  },
  {
    "name": "interval",
    "type": "string",
    "default": "\"1s\"",
    "description": "time interval for recheck"
  },
  {
    "name": "path",
    "type": "string",
    "default": "\"\"",
    "description": "URL path with query on backend host"
  },
  {
    "name": "timeout",
    "type": "string",
    "default": "\"1s\"",
    "description": "maximum allowed time limit which is\tbounded by `interval`"
  }
]

---
::
