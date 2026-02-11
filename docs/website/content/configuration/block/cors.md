---
title: 'CORS'
slug: 'cors'
---

# CORS

The `cors` block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

| Block name | Context                                                                                                                                                               | Label    |
|:-----------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------|
| `cors`     | [Server Block](/configuration/block/server), [Files Block](/configuration/block/files), [SPA Block](/configuration/block/spa), [API Block](/configuration/block/api). | no label |

**Note:** `Access-Control-Allow-Methods` is only sent in response to a CORS preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method (see the `allowed_method` attribute for [`api`](/configuration/block/api) or [`endpoint`](/configuration/block/endpoint) blocks).

### Attribute `allowed_origins`

Can be either of: a string with a single specific origin, `"*"` (all origins are allowed) or an array of specific origins.

**Example:**
```hcl
allowed_origins = ["https://www.example.com", "https://www.another.host.org"]
```

{{< attributes >}}
[
  {
    "default": "false",
    "description": "Set to `true` if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` HTTP header fields).",
    "name": "allow_credentials",
    "type": "bool"
  },
  {
    "default": "",
    "description": "An allowed origin or a list of allowed origins.",
    "name": "allowed_origins",
    "type": "string or tuple"
  },
  {
    "default": "false",
    "description": "Set to `true` to disable the inheritance of CORS from parent context.",
    "name": "disable",
    "type": "bool"
  },
  {
    "default": "",
    "description": "Indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response HTTP header fields.",
    "name": "max_age",
    "type": "duration"
  }
]
{{< /attributes >}}

{{< duration >}}
