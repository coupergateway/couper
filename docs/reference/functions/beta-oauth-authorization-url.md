# beta_oauth_authorization_url (Beta)

Creates an OAuth 2.0 authorization URL from a via string `label` referenced
[OAuth2 AC Block](../blocks/beta-oauth2-ac.md) (Beta) or [OIDC Block](../blocks/beta-oidc.md)
(Beta).

## Syntax

```hcl
string beta_oauth_authorization_url(label string)
```

## Example

```hcl
beta_oauth_authorization_url("myOAuth2")
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`base64_encode`](base64-encode.md)
* &#8674; [`beta_oauth_verifier`](beta-oauth-verifier.md) (Beta)
