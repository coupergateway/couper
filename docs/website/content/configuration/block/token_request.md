---
title: 'Token Request (Beta)'
slug: 'token_request'
---

# Token Request (Beta)

The `beta_token_request` block in the [Backend Block](/configuration/block/backend) context configures a request to get a token used to authorize backend requests.

| Block name            | Context                                       | Label                                                                                                                                                                                                                                                                                      |
|:----------------------|:----------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `beta_token_request`  | [Backend Block](/configuration/block/backend) | &#9888; A [Token Request (Beta) Block](/configuration/block/token_request) w/o a label has an implicit label `"default"`. Only **one** [Token Request (Beta) Block](/configuration/block/token_request) w/ label `"default"` per [Backend Block](/configuration/block/backend) is allowed. |
<!-- TODO: add available http methods -->

{{< attributes >}}
[
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the token request. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "Creates implicit default `Content-Type: text/plain` header field.",
    "name": "body",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a `beta_backend_token_request` error can be handled with an `error_handler`.",
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
    "description": "Sets the given request HTTP header fields.",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Creates implicit default `Content-Type: application/json` header field.",
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple"
  },
  {
    "default": "\"GET\"",
    "description": "The request method.",
    "name": "method",
    "type": "string"
  },
  {
    "default": "",
    "description": "Sets the URL query parameters.",
    "name": "query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "The token to be stored in `backends.<backend_name>.tokens.<token_request_name>`.",
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
    "description": "URL of the resource to request the token from. May be relative to an origin specified in a referenced or nested `backend` block.",
    "name": "url",
    "type": "string"
  }
]
{{< /attributes >}}

If the `url` attribute is specified and its value is an absolute URL, the protocol and host parts must be the same as in the value of the {origin} attribute of the used backend.

{{< blocks >}}
[
  {
    "description": "Configures a [backend](/configuration/block/backend) for the token request (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  }
]
{{< /blocks >}}
