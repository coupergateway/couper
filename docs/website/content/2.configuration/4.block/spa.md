# SPA

The `spa` blocks configure the Web serving for SPA assets. Can be defined multiple times as long as the `base_path`+`paths` is unique.

| Block name | Context                       | Label    | Nested block(s)           |
|:-----------|:------------------------------|:---------|:--------------------------|
| `spa`      | [Server Block](server) | Optional | [CORS Block](cors) |

```hcl
spa {
    base_path = "/my-app" # mounts on /my-app(/**,/special)
    bootstrap_file = "./htdocs/index.html"
    paths = ["/**", "/special"]
}
```

::attributes
---
values: [
  {
    "name": "access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Sets predefined [access control](../access-control) for `spa` block context."
  },
  {
    "name": "add_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as response headers in the client response"
  },
  {
    "name": "base_path",
    "type": "string",
    "default": "",
    "description": "Configures the path prefix for all requests."
  },
  {
    "name": "bootstrap_file",
    "type": "string",
    "default": "",
    "description": "Location of the bootstrap file."
  },
  {
    "name": "cors",
    "type": "object",
    "default": "",
    "description": "Configure [CORS](cors) settings."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "disable_access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Disables access controls by name."
  },
  {
    "name": "paths",
    "type": "tuple (string)",
    "default": "[]",
    "description": "List of SPA paths that need the bootstrap file."
  },
  {
    "name": "remove_response_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the client response"
  },
  {
    "name": "set_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set as response headers in the client response"
  }
]

---
::
