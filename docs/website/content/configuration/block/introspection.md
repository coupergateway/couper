---
title: 'Token Introspection (Beta)'
slug: 'introspection'
---

# Token Introspection (Beta)

The `beta_introspection` block configures [OAuth 2.0 Token Introspection (RFC 7662)](https://datatracker.ietf.org/doc/html/rfc7662) for a `jwt` block. It allows Couper to verify token validity with an authorization server in addition to local JWT signature validation.

| Block name           | Context                               | Label    |
|:---------------------|:--------------------------------------|:---------|
| `beta_introspection` | [JWT Block](/configuration/block/jwt) | no label |

## When to Use Token Introspection

Standard JWT validation only checks the token's signature and claims (expiry, audience, etc.) locally. This is fast but has a limitation: **tokens cannot be revoked before they expire**.

Use token introspection when you need to:

- **Detect revoked tokens** - The authorization server can mark tokens as inactive (e.g., after user logout, password change, or security incident)
- **Enforce real-time access policies** - Check current token status rather than relying solely on embedded claims
- **Support token revocation requirements** - Compliance scenarios that require immediate token invalidation

## How It Works

1. Couper first validates the JWT signature and claims locally
2. If valid, Couper calls the authorization server's introspection endpoint
3. The server responds with `{"active": true}` or `{"active": false}`
4. Access is granted only if both validations pass

## Caching

To reduce load on the authorization server, introspection responses are cached based on the `ttl` attribute:

- **Positive TTL** (e.g., `"60s"`): Responses are cached; the same token won't trigger another introspection call until the TTL expires
- **Zero or negative TTL** (e.g., `"0s"`): No caching; every request calls the introspection endpoint

Choose a TTL that balances security (shorter = faster revocation detection) against performance (longer = fewer introspection calls).

## Example

```hcl
definitions {
  jwt "my_jwt" {
    signature_algorithm = "RS256"
    key_file = "pub_key.pem"
    bearer = true

    beta_introspection {
      endpoint = "https://authorization-server.example/introspect"
      client_id = "my_resource_server"
      client_secret = env.CLIENT_SECRET
      ttl = "60s"
    }
  }
}
```

## Error Handling

When a token is valid but marked inactive by the introspection endpoint, Couper returns a `jwt_token_inactive` error. You can handle this with an [error_handler](/configuration/block/error_handler):

```hcl
jwt "my_jwt" {
  # ... jwt configuration ...

  beta_introspection {
    # ... introspection configuration ...
  }

  error_handler "jwt_token_inactive" {
    response {
      status = 401
      json_body = { error = "token_revoked", error_description = "This token has been revoked" }
    }
  }
}
```

{{< attributes >}}
[
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
{{< /attributes >}}

{{< duration >}}

{{< blocks >}}
[
  {
    "description": "Configures a [backend](/configuration/block/backend) for introspection requests (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures a [JWT signing profile](/configuration/block/jwt_signing_profile) to create a client assertion if `endpoint_auth_method` is either `\"client_secret_jwt\"` or `\"private_key_jwt\"`.",
    "name": "jwt_signing_profile"
  }
]
{{< /blocks >}}
