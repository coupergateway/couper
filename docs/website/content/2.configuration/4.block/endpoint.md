# Endpoint

`endpoint` blocks define the entry points of Couper. The required _label_
defines the path suffix for the incoming client request. Each `endpoint` block must
produce an explicit or implicit client response.

| Block name | Context                                                | Label                                                                  | Nested block(s)                                                                                                                                        |
|:-----------|:-------------------------------------------------------|:-----------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------|
| `endpoint` | [Server Block](server), [API Block](api) | &#9888; required, defines the path suffix for incoming client requests | [Proxy Block(s)](proxy),  [Request Block(s)](request), [Response Block](response), [Error Handler Block(s)](error_handler) |

## Endpoint Sequence

If `request` and/or `proxy` block definitions are sequential based on their `backend_responses.*` variable references
at load-time they will be executed sequentially. Unexpected responses can be caught with [error handling](/configuration/error-handling).

### Attribute `allowed_methods`

The default value `"*"` can be combined with additional methods. Methods are matched case-insensitively. `Access-Control-Allow-Methods` is only sent in response to a [CORS](cors) preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method.

**Example:**

```hcl
allowed_methods = ["GET", "POST"]` or `allowed_methods = ["*", "BREW"]
```

## Attribute `beta_required_permission`

Overrides `beta_required_permission` in a containing `api` block. If the value is a string, the same permission applies to all request methods. If there are different permissions for different request methods, use an object with the request methods as keys and string values. Methods not specified in this object are not permitted. `"*"` is the key for "all other standard methods". Methods other than `GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `OPTIONS` must be specified explicitly. A value `""` means "no permission required". For `api` blocks with at least two `endpoint`s, all endpoints must have either a) no `beta_required_permission` set or b) either `beta_required_permission` or `disable_access_control` set. Otherwise, a configuration error is thrown.

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
    "name": "access_control",
    "type": "tuple (string)",
    "default": "",
    "description": "Sets predefined access control for this block context."
  },
  {
    "name": "allowed_methods",
    "type": "tuple (string)",
    "default": "*",
    "description": "Sets allowed methods overriding a default set in the containing <code>api</code> block. Requests with a method that is not allowed result in an error response with a <code>405 Method Not Allowed</code> status."
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
    "name": "request_body_limit",
    "type": "string",
    "default": "64MiB",
    "description": "Configures the maximum buffer size while accessing <code>request.form_body</code> or <code>request.json_body</code> content. Valid units are: <code>KiB</code>, <code>MiB</code>, <code>GiB</code>"
  },
  {
    "name": "set_response_status",
    "type": "number",
    "default": "",
    "description": "Modifies the response status code."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "Defines log fields for custom logging"
  },
  {
    "name": "beta_required_permission",
    "type": "object",
    "default": "",
    "description": "expression evaluating to string or object (string)"
  }
]

---
::
