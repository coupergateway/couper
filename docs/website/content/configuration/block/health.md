---
title: 'Health'
slug: 'health'
---

# Health

Defines a recurring health check request for its backend. Results can be obtained via the [`backends.<label>.health` variables](/configuration/variables#backends).
Changes in health states and related requests will be logged. Default User-Agent will be `Couper / <version> health-check` if not provided
via `headers` attribute. An unhealthy backend will return with a [`backend_unhealthy`](/configuration/error-handling#api-error-types) error.

| Block name    | Context                                         | Label    |
|:--------------|:------------------------------------------------|:---------|
| `beta_health` | [`backend` block](/configuration/block/backend) | no label |

{{< attributes >}}
[
  {
    "default": "[200, 204, 301]",
    "description": "One of wanted response status codes.",
    "name": "expected_status",
    "type": "tuple (int)"
  },
  {
    "default": "",
    "description": "Text which the response body must contain.",
    "name": "expected_text",
    "type": "string"
  },
  {
    "default": "2",
    "description": "Failed checks needed to consider backend unhealthy.",
    "name": "failure_threshold",
    "type": "number"
  },
  {
    "default": "",
    "description": "Request HTTP header fields.",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "\"1s\"",
    "description": "Time interval for recheck.",
    "name": "interval",
    "type": "string"
  },
  {
    "default": "",
    "description": "URL path with query on backend host.",
    "name": "path",
    "type": "string"
  },
  {
    "default": "\"1s\"",
    "description": "Maximum allowed time limit which is\tbounded by `interval`.",
    "name": "timeout",
    "type": "string"
  }
]
{{< /attributes >}}
