# Server

The `server` block is one of the root configuration blocks of Couper's configuration file.

| Block name | Context | Label    | Nested block(s)                                                                                                                                       |
|:-----------|:--------|:---------|:------------------------------------------------------------------------------------------------------------------------------------------------------|
| `server`   | -       | optional | [CORS Block](/configuration/block/cors), [Files Block](/configuration/block/files), [SPA Block(s)](/configuration/block/spa) , [API Block(s)](/configuration/block/api), [Endpoint Block(s)](/configuration/block/endpoint) |

::attributes
---
values: [
  {
    "default": "[]",
    "description": "[access controls](../access-control) to protect the server. Inherited by nested blocks.",
    "name": "access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "key/value pairs to add as response headers in the client response",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "the path prefix for all requests",
    "name": "base_path",
    "type": "string"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "disables access controls by name",
    "name": "disable_access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "location of the error file template",
    "name": "error_file",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "list of names to remove headers from the client response",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "key/value pairs to set as response headers in the client response",
    "name": "set_response_headers",
    "type": "object"
  }
]

---
::
