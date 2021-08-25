# Error Handler Block

The Error Handler `label` specifies which [Error Type](../error-handling.md#error-types)
should be handled. Multiple labels are allowed. The `label` can be omitted to catch
all errors which are related to this [Access Control](../access-control.md) definition.
This has the same behavior as the error type `*`, that catches all errors explicitly.

| Block name      | Label               | Related blocks |
| --------------- | ------------------- | -------------- |
| `error_handler` | &#10003; (optional) | [Basic Auth Block](basic-auth.md), [JWT Block](jwt.md), [OAuth2 AC Block](beta-oauth2-ac.md) (Beta), [OIDC Block](beta-oidc.md) (Beta), [SAML Block](saml.md) |

```diff
! An "error_handler" block without a label is used as "*" (catch all handler).
```

## Nested blocks

* [Proxy Block(s)](proxy.md)
* [Request Block(s)](request.md)
* [Response Block](response.md)

## Attributes

The Error Handler block behaves like an [Endpoint Block](endpoint.md). It can have
the same [Attributes](endpoint.md#attributes) **except** the following:

* [`access_control`](../attributes.md)
* [`disable_access_control`](../attributes.md)
* [`request_body_limit`](../attributes.md)

## Example

```hcl
definitions {
  jwt "missing-source" {
    key_file = "keys/public.pem"
    signature_algorithm = "HS256"

    error_handler "jwt_token_missing" {
      error_file = "my_custom_error_file.html"
      response {
        status = 400
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Endpoint Block](endpoint.md)
* &#8674; [Files Block](files.md)
