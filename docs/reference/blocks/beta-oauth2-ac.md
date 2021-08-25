# OAuth2 AC Block (Beta)

The `beta_oauth2` block configures the
[`beta_oauth_authorization_url()` function](../functions/beta-oauth-authorization-url.md)
and an [Access Control](../access-control.md) for an OAuth2 Authorization Code Grant
Flow redirect endpoint. Like all [Access Control Types](../access-control.md#access-control-types),
the `beta_oauth2` block is defined in the [Definitions Block](definitions.md) and
can be referenced in other [Blocks](../blocks.md) by its required `label`.

| Block name    | Label               | Related blocks                      |
| ------------- | ------------------- | ----------------------------------- |
| `beta_oauth2` | &#10003; (required) | [Definitions Block](definitions.md) |

## Nested blocks

* [Backend Block](backend.md)
* [Error Handler Block](error-handler.md)

## Attributes

| Attribute                                        | Type    | Default                 | Description |
| ------------------------------------------------ | ------- | ----------------------- | ----------- |
| [`authorization_endpoint`](../attributes.md)     | string  | `""`                    | &#9888; Required. The authorization server endpoint URL used for authorization. |
| [`backend`](../attributes.md)                    | string  | `""`                    | A [Backend Block](backend.md) reference, defined in [Definitions Block](definitions.md). |
| [`client_id`](../attributes.md)                  | string  | `""`                    | &#9888; Required. The client identifier. |
| [`client_secret`](../attributes.md)              | string  | `""`                    | &#9888; Required. The client password. |
| [`grant_type`](../attributes.md)                 | string  | `""`                    | &#9888; Required. Available values: `"authorization_code"`. |
| [`redirect_uri`](../attributes.md)               | string  | `""`                    | &#9888; Required. The Couper endpoint for receiving the authorization code. |
| [`scope`](../attributes.md)                      | string  | `""`                    | A space separated list of requested scopes for the access token. |
| [`token_endpoint`](../attributes.md)             | string  | `""`                    | &#9888; Required. URL of the token endpoint at the authorization server. |
| [`token_endpoint_auth_method`](../attributes.md) | string  | `"client_secret_basic"` | Defines the method to authenticate the client at the token endpoint. |
| [`verifier_method`](../attributes.md)            | string  | `""`                    | &#9888; Required. The method to verify the integrity of the authorization code flow. Available values: `"ccm_s256"`, `"state"`. |
| [`verifier_value`](../attributes.md)             | string  | `""`                    | &#9888; Required. The value of the (unhashed) verifier, e.g. using cookie value created with [`beta_oauth_verifier()` function](../functions/beta-oauth-verifier.md). |

```diff
! To be able to execute a request the "oauth2" block needs a "backend" block or a "backend" block reference.
```

```diff
! Do not disable the peer certificate validation in the "backend" with "disable_certificate_validation"!
```

```diff
! If the "token_endpoint_auth_method" is set to "client_secret_post", the client credentials are transported in the request body. If is set to "client_secret_basic", the client credentials are transported via basic authentication.
```

```diff
! Relative URL references of the "redirect_uri" are resolved against the origin of the current request URL.
```

```diff
! If the authorization server supports the "code_challenge_method" "S256" (a.k.a. PKCE, see RFC 7636), we recommend to use the "verifier_method" "ccm_s256"`.
```

**See also:**

* [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [JWT Signing Profile Block](jwt-signing-profile.md)
* &#8674; [OAuth2 CC Block](oauth2-cc.md)
