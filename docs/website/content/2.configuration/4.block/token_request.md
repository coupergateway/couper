# Token Request (Beta)

The `beta_token_request` block in the [Backend Block](backend) context configures a request to get a token used to authorize backend requests.

| Block name            | Context                           | Label                                                                                                                                                                                                                       | Nested block(s)                                                                                                      |
|:----------------------|:----------------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------------------------------------------------------|
| `beta_token_request`  | [Backend Block](backend)          | &#9888; A [Token Request (Beta) Block](token_request) w/o a label has an implicit label `"default"`. Only **one** [Token Request (Beta) Block](token_request) w/ label `"default"` per [Backend Block](backend) is allowed. | [Backend Block](backend) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set. |
<!-- TODO: add available http methods -->

| Attribute(s)      | Type                                      | Default | Description                                                                                                                                                                                                                                                                                                                 | Characteristic(s)                                                                                                                                                        | Example                                                                |
|:------------------|:------------------------------------------|:--------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------------------------------------------|
| `backend`         | string                                    | -       | `backend` block reference, defined in [Definitions Block](definitions)                                                                                                                                                                                                                                                      | &#9888; required, if no [Backend Block](backend) is defined.                                                                                                             | `backend = "foo"`                                                      |
| `body`            | string                                    | -       | -                                                                                                                                                                                                                                                                                                                           | Creates implicit default `Content-Type: text/plain` header field.                                                                                                        | -                                                                      |
| `expected_status` | tuple (number)                            | -       | If defined, the response status code will be verified against this list of codes. If the status code is unexpected a `beta_backend_token_request` error can be handled with an `error_handler` at [API](../error-handling#api-related-error_handler) or [endpoint](../error-handling#endpoint-related-error_handler) level. | -                                                                                                                                                                        | -                                                                      |
| `form_body`       | object                                    | -       | -                                                                                                                                                                                                                                                                                                                           | Creates implicit default `Content-Type: application/x-www-form-urlencoded` header field.                                                                                 | -                                                                      |
| `headers`         | object                                    | -       | -                                                                                                                                                                                                                                                                                                                           | Same as `set_request_headers` in [Request Header](../modifiers#request-header).                                                                                          | -                                                                      |
| `json_body`       | null, bool, number, string, object, tuple | -       | -                                                                                                                                                                                                                                                                                                                           | Creates implicit default `Content-Type: text/plain` header field.                                                                                                        | -                                                                      |
| `method`          | string                                    | `"GET"` | -                                                                                                                                                                                                                                                                                                                           | -                                                                                                                                                                        | -                                                                      |
| `query_params`    | -                                         | -       | -                                                                                                                                                                                                                                                                                                                           | Same as `set_query_params` in [Query Parameter](../modifiers#query-parameter).                                                                                           | -                                                                      |
| `token`           | string                                    | -       | The token to be stored in `backends.<backend_name>.tokens.<token_request_name>`.                                                                                                                                                                                                                                            | &#9888; required.                                                                                                                                                        | `token = beta_token_response.json_body.access_token`                        |
| `ttl`             | duration                                  | -       | The time span for which the token is to be stores.                                                                                                                                                                                                                                                                          | &#9888; required.                                                                                                                                                        | `ttl = "${default(beta_token_response.json_body.expires_in, 3600) * 0.9}s"` |
| `url`             | string                                    | -       | -                                                                                                                                                                                                                                                                                                                           | If defined, the host part of the URL must be the same as the `origin` attribute of the used [Backend Block](backend) or [Backend Block Reference](backend) (if defined). | -                                                                      |

::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "",
    "description": "backend block reference is required if no backend block is defined"
  },
  {
    "name": "url",
    "type": "string",
    "default": "",
    "description": "If defined, the host part of the URL must be the same as the <code>origin</code> attribute of the <code>backend</code> block (if defined)."
  },
  {
    "name": "body",
    "type": "string",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: text/plain</code> header field"
  },
  {
    "name": "expected_status",
    "type": "tuple (int)",
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a <code>beta_backend_token_request</code> error can be handled with an <code>error_handler</code>"
  },
  {
    "name": "form_body",
    "type": "string",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: application/x-www-form-urlencoded</code> header field."
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "sets the given request headers"
  },
  {
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: application/json</code> header field"
  },
  {
    "name": "query_params",
    "type": "object",
    "default": "",
    "description": "sets the url query parameters"
  },
  {
    "name": "token",
    "type": "string",
    "default": "",
    "description": "The token to be stored in <code>backends.<backend_name>.tokens.<token_request_name></code>"
  },
  {
    "name": "ttl",
    "type": "string",
    "default": "",
    "description": "The time span for which the token is to be stores."
  }
]

---
::
