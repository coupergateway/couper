# Request

The `request` block creates and executes a request to a backend service.

> 📝 Multiple [`proxy`](/configuration/block/proxy) and `request` blocks are executed in parallel.

| Block name | Context                                         | Label                                                                                                                                                                                                                                                                                                                                            |
|:-----------|:------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `request`  | [Endpoint Block](/configuration/block/endpoint) | &#9888; A [Proxy Block](/configuration/block/proxy) or [Request Block](/configuration/block/request) w/o a label has an implicit label `"default"`. Only **one** [Proxy Block](/configuration/block/proxy) or [Request Block](/configuration/block/request) w/ label `"default"` per [Endpoint Block](/configuration/block/endpoint) is allowed. |
<!-- TODO: add available http methods -->


::attributes
---
values: [
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the request. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "Plain text request body, implicitly sets `Content-Type: text/plain` header field.",
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
    "description": "Form request body, implicitly sets `Content-Type: application/x-www-form-urlencoded` header field.",
    "name": "form_body",
    "type": "string"
  },
  {
    "default": "",
    "description": "Same as `set_request_headers` in [Modifiers - Request Header](../modifiers#request-header).",
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
    "description": "The request method.",
    "name": "method",
    "type": "string"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters for this request.",
    "name": "query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "URL of the resource to request. May be relative to an origin specified in a referenced or nested `backend` block.",
    "name": "url",
    "type": "string"
  }
]

---
::

If the `url` attribute is specified and its value is an absolute URL, the protocol and host parts must be the same as in the value of the {origin} attribute of the used backend.

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for the request (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
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

* [Sending requests](https://github.com/coupergateway/couper-examples/tree/master/custom-requests)
* [Sending JSON](https://github.com/coupergateway/couper-examples/tree/master/sending-json)
* [Sending forms](https://github.com/coupergateway/couper-examples/tree/master/sending-form)
