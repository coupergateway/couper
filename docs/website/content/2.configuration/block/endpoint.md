# Endpoint

`endpoint` blocks define the entry points of Couper. The required _label_
defines the path suffix for the incoming client request. Each `endpoint` block must
produce an explicit or implicit client response.

| Block name | Context                                                                            | Label                                                                  |
|:-----------|:-----------------------------------------------------------------------------------|:-----------------------------------------------------------------------|
| `endpoint` | [Server Block](/configuration/block/server), [API Block](/configuration/block/api) | &#9888; required, defines the path suffix for incoming client requests |

## Endpoint Sequence

If `request` and/or `proxy` block definitions are sequential based on their `backend_responses.*` variable references
at load-time they will be executed sequentially. Unexpected responses can be caught with [error handling](/configuration/error-handling).

### Attribute `allowed_methods`

The default value `"*"` can be combined with additional methods. Methods are matched case-insensitively. `Access-Control-Allow-Methods` is only sent in response to a [CORS](/configuration/block/cors) preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method.

**Example:**

```hcl
allowed_methods = ["GET", "POST"]` or `allowed_methods = ["*", "BREW"]
```

### Attribute `required_permission`

Overrides `required_permission` in a containing `api` block. If the value is a string, the same permission applies to all request methods. If there are different permissions for different request methods, use an object with the request methods as keys and string values. Methods not specified in this object are not permitted. `"*"` is the key for "all other standard methods". Methods other than `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` must be specified explicitly. A value `""` means "no permission required". For `api` blocks with at least two `endpoint`s, all endpoints must have either a) no `required_permission` set or b) either `required_permission` or `disable_access_control` set. Otherwise, a configuration error is thrown.

**Example:**

```hcl
required_permission = "read"
# or
required_permission = { post = "write", "*" = "" }
# or
required_permission = default(request.path_params.p, "not_set")
```

{{< attributes >}}
[
  {
    "default": "[]",
    "description": "Sets predefined access control for this block context.",
    "name": "access_control",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to add form parameters to the upstream request body.",
    "name": "add_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add query parameters to the upstream request URL.",
    "name": "add_query_params",
    "type": "object"
  },
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
    "default": "*",
    "description": "Sets allowed methods overriding a default set in the containing `api` block. Requests with a method that is not allowed result in an error response with a `405 Method Not Allowed` status.",
    "name": "allowed_methods",
    "type": "tuple (string)"
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
    "default": "",
    "description": "References a [`proxy` block](/configuration/block/proxy) in the [definitions](/configuration/block/definitions).",
    "name": "proxy",
    "type": "string"
  },
  {
    "default": "",
    "description": "List of names to remove form parameters from the upstream request body.",
    "name": "remove_form_params",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "List of names to remove query parameters from the upstream request URL.",
    "name": "remove_query_params",
    "type": "tuple (string)"
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
    "default": "\"64MiB\"",
    "description": "Configures the maximum buffer size while accessing `request.form_body` or `request.json_body` content. Valid units are: `KiB`, `MiB`, `GiB`.",
    "name": "request_body_limit",
    "type": "string"
  },
  {
    "default": "",
    "description": "Permission required to use this endpoint (see [error type](/configuration/error-handling#error-types) `insufficient_permissions`).",
    "name": "required_permission",
    "type": "string or object (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_query_params",
    "type": "object"
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
    "description": "Modifies the response status code.",
    "name": "set_response_status",
    "type": "number"
  }
]
{{< /attributes >}}

{{< blocks >}}
[
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  },
  {
    "description": "Configures a [proxy](/configuration/block/proxy) (zero or more).",
    "name": "proxy"
  },
  {
    "description": "Configures a [request](/configuration/block/request) (zero or more).",
    "name": "request"
  },
  {
    "description": "Configures the [response](/configuration/block/response) (zero or one).",
    "name": "response"
  }
]
{{< /blocks >}}
