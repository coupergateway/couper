# SAML Block

The `saml` block configures the [`saml_sso_url()` function](../functions/saml-sso-url.md)
and an [Access Control](../access-control.md) for a SAML Assertion Consumer Service
(ACS) endpoint. Like all [Access Control Types](../access-control.md#access-control-types),
the `saml` block is defined in the [Definitions Block](definitions.md) and can be
referenced in other [Blocks](../blocks.md) by its required `label`.

| Block name | Label               | Related blocks                      |
| ---------- | ------------------- | ----------------------------------- |
| `saml`     | &#10003; (required) | [Definitions Block](definitions.md) |

## Nested blocks

* [Error Handler Block](error-handler.md)

## Attributes

| Attribute                                                                 | Type   | Default | Description |
| ------------------------------------------------------------------------- | ------ | ------- | ----------- |
| [`array_attributes`](../attributes.md)  | [list](../config-types.md#list) | `[]`    | A list of assertion attributes that may have several values. |
| [`idp_metadata_file`](../attributes.md) | string                          | `""`    | &#9888; Required. File reference to the Identity Provider metadata XML file. |
| [`sp_acs_url`](../attributes.md)        | string                          | `""`    | &#9888; Required. The URL of the Service Provider's ACS endpoint. |
| [`sp_entity_id`](../attributes.md)      | string                          | `""`    | &#9888; Required. The Service Provider's entity ID. |

```diff
! Relative URL references of the "sp_acs_url" are resolved against the origin of the current request URL.
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Response Block](response.md)
* &#8674; [Server Block](server.md)
