# Request

The `request` block creates and executes a request to a backend service.

> 📝 Multiple [`proxy`](/configuration/block/proxy) and `request` blocks are executed in parallel.

| Block name | Context                           | Label                                                                                                                                                                                                                                                                      | Nested block(s)                                                                                                             |
|:-----------|:----------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:----------------------------------------------------------------------------------------------------------------------------|
| `request`  | [Endpoint Block](/configuration/block/endpoint) | &#9888; A [Proxy Block](/configuration/block/proxy) or [Request Block](/configuration/block/request) w/o a label has an implicit label `"default"`. Only **one** [Proxy Block](/configuration/block/proxy) or [Request Block](/configuration/block/request) w/ label `"default"` per [Endpoint Block](/configuration/block/endpoint) is allowed. | [Backend Block](/configuration/block/backend) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set. |
<!-- TODO: add available http methods -->


::attributes
---
values: [
  {
    "default": "",
    "description": "`backend` block reference, defined in [`definitions`](definitions). Required, if no [`backend` block](backend) or `url` is defined within.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "plain text request body, implicitly sets `Content-Type: text/plain` header field.",
    "name": "body",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code is not included in this list an [`unexpected_status` error](../error-handling#endpoint-error-types) will be thrown which can be handled with an [`error_handler`](../error-handling#endpoint-related-error_handler).",
    "name": "expected_status",
    "type": "tuple (int)"
  },
  {
    "default": "",
    "description": "form request body, implicitly sets `Content-Type: application/x-www-form-urlencoded` header field.",
    "name": "form_body",
    "type": "string"
  },
  {
    "default": "",
    "description": "request headers",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "JSON request body, implicitly sets `Content-Type: application/json` header field.",
    "name": "json_body",
    "type": "string"
  },
  {
    "default": "\"GET\"",
    "description": "the request method",
    "name": "method",
    "type": "string"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters for this request",
    "name": "query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "If defined, the host part of the URL must be the same as the `origin` attribute of the used backend.",
    "name": "url",
    "type": "string"
  }
]

---
::

### Examples

```hcl
request {
  url = "https://httpbin.org/anything"
  body = "foo"
}
```

* [Sending requests](https://github.com/avenga/couper-examples/tree/master/custom-requests)
* [Sending JSON](https://github.com/avenga/couper-examples/tree/master/sending-json)
* [Sending forms](https://github.com/avenga/couper-examples/tree/master/sending-form)
