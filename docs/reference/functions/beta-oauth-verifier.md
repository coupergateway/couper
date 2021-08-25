# beta_oauth_verifier (Beta)

Creates a cryptographically random key as specified in
[RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636), applicable for all verifier
methods; e.g. to be set as a cookie and read into `verifier_value`. Multiple calls of
this function in the same client request context return the same value.

## Syntax

```hcl
string beta_oauth_verifier()
```

## Example

```hcl
beta_oauth_verifier()
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`beta_oauth_authorization_url`](beta-oauth-authorization-url.md) (Beta)
* &#8674; [`coalesce`](coalesce.md)
