# Files

The `files` blocks configure the file serving. Can be defined multiple times as long as the `base_path` is unique.

| Block name | Context                                     | Label    |
|:-----------|:--------------------------------------------|:---------|
| `files`    | [Server Block](/configuration/block/server) | Optional |


::attributes
---
values: [
  {
    "default": "[]",
    "description": "Sets predefined access control for this block context.",
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
    "description": "Configures the path prefix for all requests.",
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
    "default": "",
    "description": "Location of the document root (directory).",
    "name": "document_root",
    "type": "string"
  },
  {
    "default": "",
    "description": "Location of the error file template.",
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

::blocks
---
values: [
  {
    "description": "Configures [CORS](/configuration/block/cors) settings (zero or one).",
    "name": "cors"
  }
]

---
::
