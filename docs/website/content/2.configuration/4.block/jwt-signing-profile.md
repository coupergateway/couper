# JWT Signing Profile Block

The `jwt_signing_profile` block lets you configure a JSON Web Token signing
profile for your gateway. It is referenced in the [`jwt_sign()` function](#functions)
by its required _label_.

An example can be found
[here](https://github.com/avenga/couper-examples/blob/master/creating-jwt/README.md).

| Block name            | Context                                 | Label            | Nested block(s) |
|:----------------------|:----------------------------------------|:-----------------|:----------------|
| `jwt_signing_profile` | [Definitions Block](#definitions-block) | &#9888; required | -               |

| Attribute(s)          | Type                  | Default | Description                                                                                 | Characteristic(s)                                                                                                                 | Example                                       |
|:----------------------|:----------------------|:--------|:--------------------------------------------------------------------------------------------|:----------------------------------------------------------------------------------------------------------------------------------|:----------------------------------------------|
| `key`                 | string                | -       | Private key (in PEM format) for `RS*` and `ES*` variants or the secret for `HS*` algorithm. | -                                                                                                                                 | -                                             |
| `key_file`            | string                | -       | Optional file reference instead of `key` usage.                                             | -                                                                                                                                 | -                                             |
| `signature_algorithm` | -                     | -       | -                                                                                           | &#9888; required. Valid values: `"RS256"`, `"RS384"`, `"RS512"`, `"HS256"`, `"HS384"`, `"HS512"`, `"ES256"`, `"ES384"`, `"ES512"` | -                                             |
| `ttl`                 | [duration](#duration) | -       | The token's time-to-live (creates the `exp` claim).                                         | -                                                                                                                                 | -                                             |
| `claims`              | object                | -       | Default claims for the JWT payload.                                                         | The claim values are evaluated per request.                                                                                       | `claims = { iss = "https://the-issuer.com" }` |
| `headers`             | object                | -       | Additional header fields for the JWT.                                                       | `alg` and `typ` cannot be set.                                                                                                    | `headers = { kid = "my-key-id" }`             |
