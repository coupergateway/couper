# Error Handler

The `error_handler` block lets you configure the handling of errors thrown in components configured by the parent blocks.

The error handler label specifies which [error type](/configuration/error-handling#error-types) should be handled. Multiple labels are allowed. The label can be omitted to catch all relevant errors. This has the same behavior as the error type `*`, that catches all errors explicitly.

Concerning child blocks and attributes, the `error_handler` block is similar to an [Endpoint Block](/configuration/block/endpoint).

| Block name  |Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
| `error_handler` | [API Block](/configuration/block/api), [Endpoint Block](/configuration/block/endpoint), [Basic Auth Block](/configuration/block/basic_auth), [JWT Block](/configuration/block/jwt), [OAuth2 AC (Beta) Block](/configuration/block/beta_oauth2), [OIDC Block](/configuration/block/oidc), [SAML Block](/configuration/block/saml) | optional | [Proxy Block(s)](/configuration/block/proxy),  [Request Block(s)](/configuration/block/request), [Response Block](/configuration/block/response), [Error Handler Block(s)](/configuration/block/error_handler) |

## Example

```hcl
basic_auth "ba" {
  # ...
  error_handler "basic_auth_credentials_missing" {
    response {
      status = 403
      json_body = {
        error = "forbidden"
      }
    }
  }
}
```

- [Error Handling for Access Controls](https://github.com/avenga/couper-examples/blob/master/error-handling-ba/README.md).

::attributes
---
values: [
  {
    "default": "",
    "description": "key/value pairs to add form parameters to the upstream request body",
    "name": "add_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add query parameters to the upstream request URL",
    "name": "add_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add as request headers in the upstream request",
    "name": "add_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to add as response headers in the client response",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "Location of the error file template.",
    "name": "error_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "list of names to remove form parameters from the upstream request body",
    "name": "remove_form_params",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "list of names to remove query parameters from the upstream request URL",
    "name": "remove_query_params",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "list of names to remove headers from the upstream request",
    "name": "remove_request_headers",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "list of names to remove headers from the client response",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL",
    "name": "set_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL",
    "name": "set_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "key/value pairs to set as request headers in the upstream request",
    "name": "set_request_headers",
    "type": "object"
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
    "description": "[`proxy`](proxy) block definition.",
    "name": "proxy"
  },
  {
    "description": "[`request`](request) block definition.",
    "name": "request"
  },
  {
    "description": "[`response`](response) block definition.",
    "name": "response"
  }
]

---
::
