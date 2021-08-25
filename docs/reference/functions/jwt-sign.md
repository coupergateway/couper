# jwt_sign

Creates and signs a JSON Web Token (JWT) from information from a via string `label`
referenced [JWT Signing Profile Block](../blocks/jwt-signing-profile.md) and additional
object `claims` provided as a function parameter.

## Syntax

```hcl
string jwt_sign(label string[, claims object])
```

## Example

```hcl
jwt_sign("myJWT")
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`json_encode`](json-encode.md)
* &#8674; [`merge`](merge.md)
