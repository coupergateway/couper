# Cors

The `cors` block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

| Block name | Context                                                                                                       | Label    | Nested block(s) |
|:-----------|:--------------------------------------------------------------------------------------------------------------|:---------|:----------------|
| `cors`     | [Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block). | no label | -               |

**Note:** `Access-Control-Allow-Methods` is only sent in response to a CORS preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method (see the `allowed_method` attribute for [`api`](/configuration/block/api) or [`endpoint`](/configuration/block/endpoint) blocks).

### Attribute `allowed_origins`

Can be either of: a string with a single specific origin, `"*"` (all origins are allowed) or an array of specific origins.

**Example:**
```hcl
allowed_origins = ["https://www.example.com", "https://www.another.host.org"]
```

::attributes
---
values: [
  {
    "name": "allowed_origins",
    "type": "object",
    "default": "",
    "description": "An allowed origin or a list of allowed origins."
  },
  {
    "name": "allow_credentials",
    "type": "bool",
    "default": "false",
    "description": "Set to <code>true</code> if the response can be shared with credentialed requests (containing <code>Cookie</code> or <code>Authorization</code> HTTP header fields)."
  },
  {
    "name": "disable",
    "type": "bool",
    "default": "false",
    "description": "Set to <code>true</code> to disable the inheritance of CORS from parent context."
  },
  {
    "name": "max_age",
    "type": "duration",
    "default": "",
    "description": "Indicates the time the information provided by the <code>Access-Control-Allow-Methods</code> and <code>Access-Control-Allow-Headers</code> response HTTP header fields."
  }
]

---
::

::duration
