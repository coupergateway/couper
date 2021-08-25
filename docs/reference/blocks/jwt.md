# JWT Block

The `jwt` block configures the JSON Web Token access control. Like all
[Access Control Types](../access-control.md#access-control-types), the `jwt` block
is defined in the [Definitions Block](definitions.md) and can be referenced in other
[Blocks](../blocks.md) by its required `label`.

| Block name | Label               | Related blocks                      |
| ---------- | ------------------- | ----------------------------------- |
| `jwt`      | &#10003; (required) | [Definitions Block](definitions.md) |

## Nested blocks

* [Error Handler Block](error-handler.md)

## Attributes

| Attribute                                 | Type                            | Default | Description |
| ----------------------------------------- | ------------------------------- | ------- | ----------- |
| [`claims`](../attributes.md)              | string                          | `""`    | A space separated list of claims for the JWT payload. |
| [`cookie`](../attributes.md)              | string                          | `""`    | Read `AccessToken` key to gain the token value from a cookie. Available values: `"AccessToken"` |
| [`header`](../attributes.md)              | string                          | `""`    | Implies `Bearer` if `Authorization` (case-insensitive) is used, otherwise any other HTTP header field name can be used. |
| [`key`](../attributes.md)                 | string                          | `""`    | Public key in PEM format for `RS*` variants or the secret for `HS*` algorithm. |
| [`key_file`](../attributes.md)            | string                          | `""`    | Optional file reference instead of `key` usage. |
| [`required_claims`](../attributes.md)     | [list](../config-types.md#list) | `{}`    | A space separated list of of claims that must be given for a valid token. |
| [`signature_algorithm`](../attributes.md) | string                          | `""`    | &#9888; Required. Available values: `"HS256"`, `"HS384"`, `"HS512"`, `"RS256"`, `"RS384"`, `"RS512"`. |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Files Block](files.md)
* &#8674; [JWT Signing Profile Block](jwt-signing-profile.md)
