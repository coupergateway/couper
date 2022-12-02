# Token Request (Beta)

The `beta_token_request` block in the [Backend Block](/configuration/block/backend) context configures a request to get a token used to authorize backend requests.

| Block name            | Context                           | Label                                                                                                                                                                                                                       | Nested block(s)                                                                                                      |
|:----------------------|:----------------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------------------------------------------------------|
| `beta_token_request`  | [Backend Block](/configuration/block/backend)          | &#9888; A [Token Request (Beta) Block](/configuration/block/token_request) w/o a label has an implicit label `"default"`. Only **one** [Token Request (Beta) Block](/configuration/block/token_request) w/ label `"default"` per [Backend Block](/configuration/block/backend) is allowed. | [Backend Block](/configuration/block/backend) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set. |
<!-- TODO: add available http methods -->

::attributes
---
values: [
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the token request. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "Creates implicit default `Content-Type: text/plain` header field",
    "name": "body",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a `beta_backend_token_request` error can be handled with an `error_handler`",
    "name": "expected_status",
    "type": "tuple (int)"
  },
  {
    "default": "",
    "description": "Creates implicit default `Content-Type: application/x-www-form-urlencoded` header field.",
    "name": "form_body",
    "type": "string"
  },
  {
    "default": "",
    "description": "sets the given request headers",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Creates implicit default `Content-Type: application/json` header field",
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple"
  },
  {
    "default": "",
    "description": "sets the url query parameters",
    "name": "query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "The token to be stored in `backends.<backend_name>.tokens.<token_request_name>`",
    "name": "token",
    "type": "string"
  },
  {
    "default": "",
    "description": "The time span for which the token is to be stored.",
    "name": "ttl",
    "type": "string"
  },
  {
    "default": "",
    "description": "If defined, the host part of the URL must be the same as the `origin` attribute of the `backend` block (if defined).",
    "name": "url",
    "type": "string"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for the token request. Mutually exclusive with `backend` attribute.",
    "name": "backend"
  }
]

---
::
