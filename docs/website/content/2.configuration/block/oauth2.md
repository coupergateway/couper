# OAuth2

The `oauth2` block in the [Backend Block](/configuration/block/backend) context configures an OAuth2 flow to request a bearer token for the backend request.

**Note:** The token received from the authorization server's token endpoint is stored **per backend**. So even with flows where a user's account characteristics like username/password or email address are involved, there is no way to "switch" from one user to another depending on the client request.

| Block name | Context                                       | Label    |
|:-----------|:----------------------------------------------|:---------|
| `oauth2`   | [Backend Block](/configuration/block/backend) | no label |

A nested `jwt_signing_profile` block is used in two cases:
* to create a client assertion if `token_endpoint_auth_method` is either `"client_secret_jwt"` or `"private_key_jwt"`; or
* to create an assertion if `grant_type` is `"urn:ietf:params:oauth:grant-type:jwt-bearer"` and no `assertion` attribute is set.

{{< attributes >}}
[
  {
    "default": "",
    "description": "The assertion (JWT for jwt-bearer flow). Required if `grant_type` is `\"urn:ietf:params:oauth:grant-type:jwt-bearer\"` and no nested `jwt_signing_profile` block is present.",
    "name": "assertion",
    "type": "string"
  },
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for token requests. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client identifier. Required unless the `grant_type` is `\"urn:ietf:params:oauth:grant-type:jwt-bearer\"`.",
    "name": "client_id",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client password. Required unless `token_endpoint_auth_method` is `\"private_key_jwt\"` or the `grant_type` is `\"urn:ietf:params:oauth:grant-type:jwt-bearer\"`.",
    "name": "client_secret",
    "type": "string"
  },
  {
    "default": "",
    "description": "Required, valid values: `\"client_credentials\"`, `\"password\"`, `\"urn:ietf:params:oauth:grant-type:jwt-bearer\"`.",
    "name": "grant_type",
    "type": "string"
  },
  {
    "default": "",
    "description": "The (service account's) password (for password flow). Required if grant_type is `\"password\"`.",
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
    "description": "The (service account's) username (for password flow). Required if grant_type is `\"password\"`.",
    "name": "username",
    "type": "string"
  }
]
{{< /attributes >}}

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](/configuration/modifiers#request-header) in a [backend block](/configuration/block/backend).

{{< blocks >}}
[
  {
    "description": "Configures a [backend](/configuration/block/backend) for token requests (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures a [JWT signing profile](/configuration/block/jwt_signing_profile) to create a client assertion if `token_endpoint_auth_method` is either `\"client_secret_jwt\"` or `\"private_key_jwt\"`, or to create an assertion if `grant_type` is `\"urn:ietf:params:oauth:grant-type:jwt-bearer\"` and no `assertion` attribute is set (zero or one).",
    "name": "jwt_signing_profile"
  }
]
{{< /blocks >}}
