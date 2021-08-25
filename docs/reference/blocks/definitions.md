# Definitions Block

The `definitions` block contains reusable (e.g. [Backend Block](backend.md)) or
[Access Control](../access-control.md) configurations.

| Block name    | Label    | Related blocks |
| ------------- | :------: | :------------: |
| `definitions` | &#10005; | &#10005;       |

```diff
! The access control blocks are exclusively defined in the "definitions" block.
```

**See also:**

* [Access Control](../access-control.md)

## Nested blocks

* [Backend Block(s)](backend.md)
* [Basic Auth Block(s)](basic-auth.md)
* [JWT Block(s)](jwt.md)
* [JWT Signing Profile Block(s)](jwt-signing-profile.md)
* [OAuth2 AC Block(s)](beta-oauth2-ac.md) (Beta)
* [OIDC Block(s)](beta-oidc.md) (Beta)
* [SAML Block(s)](saml.md)

## Examples

* [Securing APIs](../examples.md#securing-apis)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Defaults Block](defaults.md)
* &#8674; [Endpoint Block](endpoint.md)
