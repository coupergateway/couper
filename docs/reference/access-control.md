# Configuration Reference ~ Access Control

The configuration of the Access Control is twofold in Couper:

1. The [Access Control Type](#access-control-types) has to be predefined in the
[Definitions Block](blocks/definitions.md), each with a unique label.
2. Anywhere in a [Protectable Block](#protectable-blocks) those labels can be used
in the [`access_control`](attributes.md) list attribute to protect that block.

```diff
! Note, all access rights are inherited by the nested blocks.
```

Each Access Control can also be deactivated by typing those label in the
[`disable_access_control`](attributes.md) list attribute. See the
[example](#examples) below.

```diff
! Note, the "disable_access_control" has priority over the "access_control" list attribute.
```

All [Access Control Types](#access-control-types) support the
[Error Handling](error-handling.md) to handle related errors.

## Access Control Types

The following types can be used to protect the [Protectable Block](#protectable-blocks):

* [Basic Auth Block](blocks/basic-auth.md)
* [JWT Block](blocks/jwt.md)
* [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta)
* [OIDC Block](blocks/beta-oidc.md) (Beta)
* [SAML Block](blocks/saml.md)

## Protectable Blocks

The following blocks are protectable by all [Access Control Types](#access-control-types):

* [API Block](blocks/api.md)
* [Endpoint Block](blocks/endpoint.md)
* [Files Block](blocks/files.md)
* [Server Block](blocks/server.md)
* [SPA Block](blocks/spa.md)

## Examples

* [Securing APIs](examples.md#securing-apis)

Refer to the comments in the following example configuration:

```hcl
definitions {
  // Define a "basic_auth" Aceess Control named "BA-AC"
  basic_auth "BA-AC" {...}

  // Define a "saml" Aceess Control named "SAML-AC"
  saml "SAML-AC" {...}
}

server "example" {
  // Both, BA-AC and SAML-AC protect all endpoints in this example configuration.
  access_control = ["BA-AC", "SAML-AC"]

  endpoint "/insecure" {
    // Disables both, BA-AC and SAML-AC for the current endpoint.
    disable_access_control = ["BA-AC", "SAML-AC"]
    ...
  }
  endpoint "/secure" {
    // BA-AC is disabled, but SAML-AC protects farther the current endpoint.
    disable_access_control = ["BA-AC"]
    ...
  }
  endpoint "/super-secure" {
    // Both, BA-AC and SAML-AC protect the current endpoint.
    ...
  }
}
```

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Variables](variables.md)
* &#8674; [Attributes](attributes.md)
