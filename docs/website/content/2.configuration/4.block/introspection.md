# Token Introspection (Beta)

The `introspection` block lets you configure OAuth2 token introspection for an encapsulating `jwt` block.

| Block name           | Context                               | Label    |
|:---------------------|:--------------------------------------|:---------|
| `beta_introspection` | [JWT Block](/configuration/block/jwt) | no label |

::attributes
---
values: [
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for introspection requests. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client identifier.",
    "name": "client_id",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client password. Required unless the `endpoint_auth_method` is `\"private_key_jwt\"`.",
    "name": "client_secret",
    "type": "string"
  },
  {
    "default": "",
    "description": "The authorization server's `introspection_endpoint`.",
    "name": "endpoint",
    "type": "string"
  },
  {
    "default": "\"client_secret_basic\"",
    "description": "Defines the method to authenticate the client at the introspection endpoint. If set to `\"client_secret_post\"`, the client credentials are transported in the request body. If set to `\"client_secret_basic\"`, the client credentials are transported via Basic Authentication. If set to `\"client_secret_jwt\"`, the client is authenticated via a JWT signed with the `client_secret`. If set to `\"private_key_jwt\"`, the client is authenticated via a JWT signed with its private key (see `jwt_signing_profile` block).",
    "name": "endpoint_auth_method",
    "type": "string"
  },
  {
    "default": "",
    "description": "The time-to-live of a cached introspection response. With a non-positive value the introspection endpoint is called each time a token is validated.",
    "name": "ttl",
    "type": "duration"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for introspection requests (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures a [JWT signing profile](/configuration/block/jwt_signing_profile) to create a client assertion if `endpoint_auth_method` is either `\"client_secret_jwt\"` or `\"private_key_jwt\"`.",
    "name": "jwt_signing_profile"
  }
]

---
::
