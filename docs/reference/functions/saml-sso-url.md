# saml_sso_url

Creates a SAML SingleSignOn URL (including the `SAMLRequest` parameter) from a
via string `label` referenced [SAML Block](../blocks/saml.md).

## Syntax

```hcl
string saml_sso_url(label string)
```

## Example

```hcl
saml_sso_url("mySAML")
```

-----

## Navigation

* &#8673; [Functions](../functions.md)
* &#8672; [`merge`](merge.md)
* &#8674; [`to_lower`](to-lower.md)
