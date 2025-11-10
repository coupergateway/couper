# WebSockets

The `websockets` block activates support for WebSocket connections in Couper.

| Block name   | Context                               | Label    |
|:-------------|:--------------------------------------|:---------|
| `websockets` | [`proxy`](/configuration/block/proxy) | no label |

{{< attributes >}}
[
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
    "default": "",
    "description": "The total deadline [duration](#duration) a WebSocket connection has to exist.",
    "name": "timeout",
    "type": "string"
  }
]
{{< /attributes >}}

{{< duration >}}
