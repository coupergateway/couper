# OAuth2

The `oauth2` block in the [Backend Block](backend) context configures an OAuth2 flow to request a bearer token for the backend request.

**Note:** The token received from the authorization server's token endpoint is stored **per backend**. So even with flows where a user's account characteristics like username/password or email address are involved, there is no way to "switch" from one user to another depending on the client request.

| Block name | Context                  | Label    | Nested block(s)                                                            |
|:-----------|:-------------------------|:---------|:---------------------------------------------------------------------------|
| `oauth2`   | [Backend Block](backend) | no label | [Backend Block](backend), [JWT Signing Profile Block](jwt_signing_profile) |

::attributes
---
values: [
  {
    "default": "",
    "description": "The assertion (JWT for jwt-bearer flow). Required if `grant_type` is `urn:ietf:params:oauth:grant-type:jwt-bearer`.",
    "name": "assertion",
    "type": "string"
  },
  {
    "default": "",
    "description": "[`backend` block](backend) reference.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client identifier. Required unless the `grant_type` is `urn:ietf:params:oauth:grant-type:jwt-bearer`.",
    "name": "client_id",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client password. Required unless the `grant_type` is `urn:ietf:params:oauth:grant-type:jwt-bearer`.",
    "name": "client_secret",
    "type": "string"
  },
  {
    "default": "",
    "description": "Required, valid values: `client_credentials`, `password`, `urn:ietf:params:oauth:grant-type:jwt-bearer`",
    "name": "grant_type",
    "type": "string"
  },
  {
    "default": "",
    "description": "The (service account's) password (for password flow). Required if grant_type is `password`.",
    "name": "password",
    "type": "string"
  },
  {
    "default": "1",
    "description": "The number of retries to get the token and resource, if the resource-request responds with `401 Unauthorized` HTTP status code.",
    "name": "retries",
    "type": "number"
  },
  {
    "default": "",
    "description": "A space separated list of requested scope values for the access token.",
    "name": "scope",
    "type": "string"
  },
  {
    "default": "",
    "description": "URL of the token endpoint at the authorization server.",
    "name": "token_endpoint",
    "type": "string"
  },
  {
    "default": "\"client_secret_basic\"",
    "description": "Defines the method to authenticate the client at the token endpoint. If set to `\"client_secret_post\"`, the client credentials are transported in the request body. If set to `\"client_secret_basic\"`, the client credentials are transported via Basic Authentication. If set to `\"client_secret_jwt\"`, the client is authenticated via a JWT signed with the `client_secret`. If set to `\"private_key_jwt\"`, the client is authenticated via a JWT signed with its private key (see `jwt_signing_profile` block).",
    "name": "token_endpoint_auth_method",
    "type": "string"
  },
  {
    "default": "",
    "description": "The (service account's) username (for password flow). Required if grant_type is `password`.",
    "name": "username",
    "type": "string"
  }
]

---
::

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](../modifiers#request-header) in a [backend block](backend).
