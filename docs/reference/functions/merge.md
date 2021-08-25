# merge

Deep-merges two or more of either objects or [tuples](../config-types.md#tuple) `arg`
parameters. `null` arguments are ignored. A `null` attribute value in an object
removes the previous attribute value. An attribute value with a different type than
the current value is set as the new value. `merge()` without parameters returns
`null`.

## Syntax

```hcl
(object|tuple) merge(arg... (object|tuple))
```

## Example

```hcl
merge(request.headers, { x-additional = "myVal" })
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`jwt_sign`](jwt-sign.md)
* &#8674; [`saml_sso_url`](saml-sso-url.md)
