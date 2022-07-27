# SAML

The `saml` block lets you configure the `saml_sso_url()` function](../functions) and an access
control for a SAML Assertion Consumer Service (ACS) endpoint.
Like all [Access Control](#access-control) types, the `saml` block is defined in
the [Definitions Block](definitions) and can be referenced in all configuration blocks by its
required _label_.

| Block name | Context                                 | Label            | Nested block(s)                             |
|:-----------|:----------------------------------------|:-----------------|:--------------------------------------------|
| `saml`     | [Definitions Block](definitions) | &#9888; required | [Error Handler Block](error_handler) |

| Attribute(s)        | Type           | Default | Description                                                      | Characteristic(s)                                                                                                                                                                                                                 | Example                           |
|:--------------------|:---------------|:--------|:-----------------------------------------------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:----------------------------------|
| `idp_metadata_file` | string         | -       | File reference to the Identity Provider metadata XML file.       | &#9888; required                                                                                                                                                                                                                  | -                                 |
| `sp_acs_url`        | string         | -       | The URL of the Service Provider's ACS endpoint.                  | &#9888; required. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [`accept_forwarded_url`](settings) attribute if Couper is running behind a proxy. | -                                 |
| `sp_entity_id`      | string         | -       | The Service Provider's entity ID.                                | &#9888; required                                                                                                                                                                                                                  | -                                 |
| `array_attributes`  | tuple (string) | `[]`    | A list of assertion attributes that may have several values.     | Results in at least an empty array in `request.context.<label>.attributes.<name>`                                                                                                                                                 | `array_attributes = ["memberOf"]` |
| `custom_log_fields` | object         | -       | Defines log fields for [Custom Logging](/observation/logging#custom-logging). | &#9888; Inherited by nested blocks.                                                                                                                                                                                               | -                                 |

Some information from the assertion consumed at the ACS endpoint is provided in the context at `request.context.<label>`:

- the `NameID` of the assertion's `Subject` (`request.context.<label>.sub`)
- the session expiry date `SessionNotOnOrAfter` (as UNIX timestamp: `request.context.<label>.exp`)
- the attributes (`request.context.<label>.attributes.<name>`)
