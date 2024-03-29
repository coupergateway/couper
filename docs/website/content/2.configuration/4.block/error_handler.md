# Error Handler

The `error_handler` block lets you configure the handling of errors thrown in components configured by the parent blocks.

The error handler label specifies which [error type](/configuration/error-handling#error-types) should be handled. Multiple labels are allowed. The label can be omitted to catch all relevant errors. This has the same behavior as the error type `*`, that catches all errors explicitly.

Concerning child blocks and attributes, the `error_handler` block is similar to an [Endpoint Block](/configuration/block/endpoint).

| Block name      | Context                                                                                                                                                                                                                                                                                                                          | Label    |
| :---------------| :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------| :--------|
| `error_handler` | [API Block](/configuration/block/api), [Endpoint Block](/configuration/block/endpoint), [Basic Auth Block](/configuration/block/basic_auth), [JWT Block](/configuration/block/jwt), [OAuth2 AC (Beta) Block](/configuration/block/beta_oauth2), [OIDC Block](/configuration/block/oidc), [SAML Block](/configuration/block/saml) | optional |

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

- [Error Handling for Access Controls](https://github.com/coupergateway/couper-examples/blob/master/error-handling-ba/README.md).

::attributes
---
values: [
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
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
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
  }
]

---
::

::blocks
---
values: [
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

---
::
