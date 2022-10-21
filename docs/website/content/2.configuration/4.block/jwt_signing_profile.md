# JWT Signing Profile

The `jwt_signing_profile` block lets you configure a JSON Web Token signing
profile for your gateway. It is referenced in the [`jwt_sign()` function](../functions)
by its required _label_.

It can also be used (without _label_) in [`oauth2`](oauth2), [`oidc`](oidc) or
[`beta_oauth2`](beta_oauth2) blocks for `token_endpoint_auth_method`s `"client_secret_jwt"`
or `"private_key_jwt"`.

| Block name            | Context                                                                                                             | Label                              | Nested block(s) |
|:----------------------|:--------------------------------------------------------------------------------------------------------------------|:-----------------------------------|:----------------|
| `jwt_signing_profile` | [Definitions Block](definitions), [OAuth2 Block](oauth2), [OAuth2 AC (Beta) Block](beta_oauth2), [OIDC Block](oidc) | required if defined in defititions | -               |


::attributes
---
values: [
  {
    "default": "",
    "description": "claims for the JWT payload, claim values are evaluated per request",
    "name": "claims",
    "type": "object"
  },
  {
    "default": "",
    "description": "additional header fields for the JWT, `alg` and `typ` cannot be set",
    "name": "headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "private key (in PEM format) for `RS*` and `ES*` variants or the secret for `HS*` algorithms",
    "name": "key",
    "type": "string"
  },
  {
    "default": "",
    "description": "optional file reference instead of `key` usage",
    "name": "key_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "algorithm used for signing: `\"RS256\"`, `\"RS384\"`, `\"RS512\"`, `\"HS256\"`, `\"HS384\"`, `\"HS512\"`, `\"ES256\"`, `\"ES384\"`, `\"ES512\"`",
    "name": "signature_algorithm",
    "type": "string"
  },
  {
    "default": "",
    "description": "The token's time-to-live, creates the `exp` claim",
    "name": "ttl",
    "type": "string"
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
