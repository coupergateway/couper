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

### Using a metadata file

```hcl
saml "SSO" {
  idp_metadata_file = "idp-metadata.xml"
  sp_entity_id = env.SP_ENTITY_ID
  sp_acs_url = "http://localhost:8080/saml/acs"
  array_attributes = ["eduPersonAffiliation"] # or ["memberOf"]
}
```

### Using a metadata URL with automatic refresh

```hcl
saml "SSO" {
  idp_metadata_url = "https://idp.example.com/metadata"
  metadata_ttl = "1h"
  metadata_max_stale = "1h"
  sp_entity_id = env.SP_ENTITY_ID
  sp_acs_url = "http://localhost:8080/saml/acs"
  array_attributes = ["eduPersonAffiliation"]
}
```

### Using a metadata URL with a custom backend

```hcl
saml "SSO" {
  idp_metadata_url = "https://idp.example.com/metadata"
  metadata_ttl = "30m"
  sp_entity_id = env.SP_ENTITY_ID
  sp_acs_url = "http://localhost:8080/saml/acs"

  backend {
    origin = "https://idp.example.com"
    timeout = "10s"
  }
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
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for IdP metadata requests. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "File reference to the Identity Provider metadata XML file. Mutually exclusive with `idp_metadata_url`.",
    "name": "idp_metadata_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "URL to fetch the Identity Provider metadata XML. Mutually exclusive with `idp_metadata_file`.",
    "name": "idp_metadata_url",
    "type": "string"
  },
  {
    "default": "\"1h\"",
    "description": "Time period the cached IdP metadata stays valid after its TTL has passed.",
    "name": "metadata_max_stale",
    "type": "duration"
  },
  {
    "default": "\"1h\"",
    "description": "Time period the IdP metadata stays valid and may be cached.",
    "name": "metadata_ttl",
    "type": "duration"
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
    "description": "Configures a [backend](/configuration/block/backend) for IdP metadata requests. Mutually exclusive with `backend` attribute.",
    "name": "backend"
  },
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]
{{< /blocks >}}
