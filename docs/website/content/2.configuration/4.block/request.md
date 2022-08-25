# Request

The `request` block creates and executes a request to a backend service.

> 📝 Multiple [`proxy`](proxy) and `request` blocks are executed in parallel.

| Block name | Context                           | Label                                                                                                                                                                                                                                                                      | Nested block(s)                                                                                                             |
|:-----------|:----------------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:----------------------------------------------------------------------------------------------------------------------------|
| `request`  | [Endpoint Block](endpoint) | &#9888; A [Proxy Block](proxy) or [Request Block](request) w/o a label has an implicit label `"default"`. Only **one** [Proxy Block](proxy) or [Request Block](request) w/ label `"default"` per [Endpoint Block](endpoint) is allowed. | [Backend Block](backend) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set. |
<!-- TODO: add available http methods -->


::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "",
    "description": "`backend` block reference, defined in [`definitions`](definitions). Required, if no [`backend` block](backend) or `url` is defined within."
  },
  {
    "name": "body",
    "type": "string",
    "default": "",
    "description": "plain text request body, implicitly sets `Content-Type: text/plain` header field."
  },
  {
    "name": "expected_status",
    "type": "tuple (int)",
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of codes. If the status code is not included in this list an [`unexpected_status` error](../error-handling#endpoint-error-types) will be thrown which can be handled with an [`error_handler`](../error-handling#endpoint-related-error_handler)."
  },
  {
    "name": "form_body",
    "type": "string",
    "default": "",
    "description": "form request body, implicitly sets `Content-Type: application/x-www-form-urlencoded` header field."
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "request headers"
  },
  {
    "name": "json_body",
    "type": "string",
    "default": "",
    "description": "JSON request body, implicitly sets `Content-Type: application/json` header field."
  },
  {
    "name": "method",
    "type": "string",
    "default": "\"GET\"",
    "description": "the request method"
  },
  {
    "name": "query_params",
    "type": "object",
    "default": "",
    "description": "Key/value pairs to set query parameters for this request"
  },
  {
    "name": "url",
    "type": "string",
    "default": "",
    "description": "If defined, the host part of the URL must be the same as the `origin` attribute of the used backend."
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
