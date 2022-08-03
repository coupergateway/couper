# JWT Signing Profile

The `jwt_signing_profile` block lets you configure a JSON Web Token signing
profile for your gateway. It is referenced in the [`jwt_sign()` function](../functions)
by its required _label_.

| Block name            | Context                                 | Label            | Nested block(s) |
|:----------------------|:----------------------------------------|:-----------------|:----------------|
| `jwt_signing_profile` | [Definitions Block](definitions)        | required         | -               |


::attributes
---
values: [
  {
    "name": "claims",
    "type": "object",
    "default": "",
    "description": "claims for the JWT payload, claim values are evaluated per request"
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "additional header fields for the JWT, `alg` and `typ` cannot be set"
  },
  {
    "name": "key",
    "type": "string",
    "default": "",
    "description": "private key (in PEM format) for `RS*` and `ES*` variants or the secret for `HS*` algorithms"
  },
  {
    "name": "key_file",
    "type": "string",
    "default": "",
    "description": "optional file reference instead of `key` usage"
  },
  {
    "name": "signature_algorithm",
    "type": "string",
    "default": "",
    "description": "algorithm used for signing: `\"RS256\"`, `\"RS384\"`, `\"RS512\"`, `\"HS256\"`, `\"HS384\"`, `\"HS512\"`, `\"ES256\"`, `\"ES384\"`, `\"ES512\"`"
  },
  {
    "name": "ttl",
    "type": "string",
    "default": "",
    "description": "The token's time-to-live, creates the `exp` claim"
  }
]

---
::


::duration

---
::

### Example

```hcl
jwt_signing_profile "myjwt" {
  signature_algorithm = "RS256"
  key_file = "priv_key.pem"
  ttl = "600s"
  claims = {
    iss = "MyAS"
    iat = unixtime()
  }
  headers = {
    kid = "my-jwk-id"
  }
}
```

A detailed example can be found [here](https://github.com/avenga/couper-examples/blob/master/creating-jwt/README.md).
