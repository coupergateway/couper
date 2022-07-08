# Api

The `api` block bundles endpoints under a certain `base_path`.

> If an error occurred for api endpoints the response gets processed
as JSON error with an error body payload. This can be customized via `error_file`.

| Block name | Context                       | Label    | Nested block(s)                                                                                                 |
|:-----------|:------------------------------|:---------|:----------------------------------------------------------------------------------------------------------------|
| `api`      | [Server Block](#server-block) | Optional | [Endpoint Block(s)](/configuration/block/endpoint), [CORS Block](/configuration/block/cors), [Error Handler Block(s)](/configuration/error-handling) |


### Attribute `allowed_methods`

The default value `*` can be combined with additional methods. Methods are matched case-insensitively. `Access-Control-Allow-Methods` is only sent in response to a [CORS](/configuration/block/cors) preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method.

**Example:** `allowed_methods = ["GET", "POST"]` or `allowed_methods = ["*", "BREW"]`

::attributes
---
values: [
  {
    "name": "access_control",
    "type": "tuple (string)",
    "default": "",
    "description": "Sets predefined [Access Control](#access-control) for this block."
  },
  {
    "name": "allowed_methods",
    "type": "tuple (string)",
    "default": "*",
    "description": "Sets allowed methods as _default_ for all contained endpoints. Requests with a method that is not allowed result in an error response with a <code>405 Method Not Allowed</code> status."
  },
  {
    "name": "base_path",
    "type": "string",
    "default": "",
    "description": "Configures the path prefix for all requests."
  },
  {
    "name": "disable_access_control",
    "type": "tuple (string)",
    "default": "",
    "description": "Disables access controls by name."
  },
  {
    "name": "error_file",
    "type": "string",
    "default": "",
    "description": "Location of the error file template."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "Defines log fields for custom Logging"
  },
  {
    "name": "beta_required_permission",
    "type": "object",
    "default": "",
    "description": "Permission required to use this API (see [error type](ERRORS.md#error-types) <code>beta_insufficient_permissions)</code>."
  }
]

---
::
