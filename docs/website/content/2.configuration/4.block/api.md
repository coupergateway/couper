# API

The `api` block bundles endpoints under a certain `base_path`.

> If an error occurred for api endpoints the response gets processed
as JSON error with an error body payload. This can be customized via `error_file`.

| Block name | Context                       | Label    | Nested block(s)                                                                                                 |
|:-----------|:------------------------------|:---------|:----------------------------------------------------------------------------------------------------------------|
| `api`      | [Server Block](/configuration/block/server) | Optional | [Endpoint Block(s)](/configuration/block/endpoint), [CORS Block](/configuration/block/cors), [Error Handler Block(s)](/configuration/block/error_handler) |


### Attribute `allowed_methods`

The default value `*` can be combined with additional methods. Methods are matched case-insensitively. `Access-Control-Allow-Methods` is only sent in response to a [CORS](/configuration/block/cors) preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method.

**Example:** `allowed_methods = ["GET", "POST"]` or `allowed_methods = ["*", "BREW"]`


### Attribute `beta_required_permission`

If the value is a string, the same permission applies to all request methods. If there are different permissions for different request methods, use an object with the request methods as keys and string values. Methods not specified in this object are not permitted. `"*"` is the key for "all other standard methods". Methods other than `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` must be specified explicitly. A value `""` means "no permission required".

**Example:**

```hcl
beta_required_permission = "read"
# or
beta_required_permission = { post = "write", "*" = "" }
# or
beta_required_permission = default(request.path_params.p, "not_set")
```

::attributes
---
values: [
  {
    "default": "[]",
    "description": "Sets predefined [access control](../access-control) for this block.",
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
    "default": "*",
    "description": "Sets allowed methods as _default_ for all contained endpoints. Requests with a method that is not allowed result in an error response with a `405 Method Not Allowed` status.",
    "name": "allowed_methods",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Configures the path prefix for all requests.",
    "name": "base_path",
    "type": "string"
  },
  {
    "default": "",
    "description": "Permission required to use this API (see [error type](/configuration/error-handling#error-types) `beta_insufficient_permissions`).",
    "name": "beta_required_permission",
    "type": "string or object (string)"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
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
values: null

---
::
