---
title: 'SAML'
slug: 'saml'
---

# SAML

The `saml` block lets you configure the [`saml_sso_url()` function](/configuration/functions) and an access
control for a SAML Assertion Consumer Service (ACS) endpoint.
Like all [access control](/configuration/access-control) types, the `saml` block is defined in
the [`definitions` block](/configuration/block/definitions) and can be referenced in all configuration blocks by its
required _label_.

| Block name | Context                                               | Label            |
|:-----------|:------------------------------------------------------|:-----------------|
| `saml`     | [Definitions Block](/configuration/block/definitions) | &#9888; required |

## Example

A complete example can be found [here](https://github.com/coupergateway/couper-examples/tree/master/saml).

```hcl
saml "SSO" {
  idp_metadata_file = "idp-metadata.xml"
  sp_entity_id = env.SP_ENTITY_ID
  sp_acs_url = "http://localhost:8080/saml/acs"
  array_attributes = ["eduPersonAffiliation"] # or ["memberOf"]
}
```


{{< attributes >}}
[
  {
    "default": "[]",
    "description": "A list of assertion attributes that may have several values. Results in at least an empty array in `request.context.<label>.attributes.<name>`",
    "name": "array_attributes",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "File reference to the Identity Provider metadata XML file.",
    "name": "idp_metadata_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "The URL of the Service Provider's ACS endpoint. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [`accept_forwarded_url` attribute](settings) if Couper is running behind a proxy.",
    "name": "sp_acs_url",
    "type": "string"
  },
  {
    "default": "",
    "description": "The Service Provider's entity ID.",
    "name": "sp_entity_id",
    "type": "string"
  }
]
{{< /attributes >}}

Some information from the assertion consumed at the ACS endpoint is provided in the context at `request.context.<label>`:

  - the `NameID` of the assertion's `Subject` (`request.context.<label>.sub`)
  - the session expiry date `SessionNotOnOrAfter` (as UNIX timestamp: `request.context.<label>.exp`)
  - the attributes (`request.context.<label>.attributes.<name>`)

{{< blocks >}}
[
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]
{{< /blocks >}}
