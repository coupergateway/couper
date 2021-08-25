# coalesce

Returns the first of the given various `arg` that is not null.

## Syntax

```hcl
various coalesce(arg... various)
```

## Example

```hcl
coalesce(request.cookies.foo, "bar")
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`beta_oauth_verifier`](beta-oauth-verifier.md) (Beta)
* &#8674; [`json_decode`](json-decode.md)
