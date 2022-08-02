# Files

The `files` blocks configure the file serving. Can be defined multiple times as long as the `base_path` is unique.

| Block name | Context                       | Label    | Nested block(s)           |
|:-----------|:------------------------------|:---------|:--------------------------|
| `files`    | [Server Block](server) | Optional | [CORS Block](cors) |


::attributes
---
values: [
  {
    "name": "access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Sets predefined access control for this block context."
  },
  {
    "name": "base_path",
    "type": "string",
    "default": "",
    "description": "Configures the path prefix for all requests."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "Defines log fields for custom logging."
  },
  {
    "name": "document_root",
    "type": "string",
    "default": "",
    "description": "Location of the document root (directory)."
  },
  {
    "name": "error_file",
    "type": "string",
    "default": "",
    "description": "Location of the error file template."
  }
]

---
::
