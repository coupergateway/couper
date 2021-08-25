# JWT Signing Profile Block

The `jwt_signing_profile` block configures a JSON Web Token signing profile. It
is referenced in the [`jwt_sign()` function](../functions/jwt-sign.md) by its required
`label`.

| Block name            | Label               | Related blocks                      |
| --------------------- | ------------------- | ----------------------------------- |
| `jwt_signing_profile` | &#10003; (required) | [Definitions Block](definitions.md) |

## Attributes

| Attribute                                 | Type                                    | Default  | Description |
| ----------------------------------------- | --------------------------------------- | -------- | ----------- |
| [`claims`](../attributes.md)              | string                                  | `""`     | A space separated list of claims for the JWT payload. |
| [`key`](../attributes.md)                 | string                                  | `""`     | Private key in PEM format for `RS*` variants or the secret for `HS*` algorithm. |
| [`key_file`](../attributes.md)            | string                                  | `""`     | Optional file reference instead of `key` usage. |
| [`signature_algorithm`](../attributes.md) | string                                  | `""`     | &#9888; Required. Available values: `"HS256"`, `"HS384"`, `"HS512"`, `"RS256"`, `"RS384"`, `"RS512"`. |
| [`ttl`](../attributes.md)                 | [duration](../config-types.md#duration) | &#10005; | The token's time-to-live (creates the `exp` claim). |

## Examples

* [Creating JWT](https://github.com/avenga/couper-examples/blob/master/creating-jwt/README.md)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [JWT Block](jwt.md)
* &#8674; [OAuth2 AC Block](beta-oauth2-ac.md) (Beta)
