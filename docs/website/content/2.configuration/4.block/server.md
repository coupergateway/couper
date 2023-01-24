# Server

The `server` block is one of the root configuration blocks of Couper's configuration file.

| Block name | Context | Label    |
|:-----------|:--------|:---------|
| `server`   | -       | optional |

::attributes
---
values: [
  {
    "default": "[]",
    "description": "The [access controls](../access-control) to protect the server. Inherited by nested blocks.",
    "name": "access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as response headers in the client response.",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "The path prefix for all requests.",
    "name": "base_path",
    "type": "string"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "Disables access controls by name.",
    "name": "disable_access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Location of the error file template.",
    "name": "error_file",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the client response.",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as response headers in the client response.",
    "name": "set_response_headers",
    "type": "object"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures an API (zero or more).",
    "name": "api"
  },
  {
    "description": "Configures [CORS](/configuration/block/cors) settings (zero or one).",
    "name": "cors"
  },
  {
    "description": "Configures a free [endpoint](/configuration/block/endpoint) (zero or more).",
    "name": "endpoint"
  },
  {
    "description": "Configures file serving (zero or more).",
    "name": "files"
  },
  {
    "description": "Configures an SPA (zero or more).",
    "name": "spa"
  },
  {
    "description": "Configures [server TLS](/configuration/block/server_tls) (zero or one).",
    "name": "tls"
  }
]

---
::
