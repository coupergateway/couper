# OAuth2 Client Credentials Block

The `oauth2` block in the [Backend Block](backend.md) context configures the OAuth2
Client Credentials Flow to request a bearer token for the backend request.

| Block name | Label    | Related blocks              |
| ---------- | :------: | --------------------------- |
| `oauth2`   | &#10005; | [Backend Block](backend.md) |

## Nested blocks

* [Backend Block](backend.md)

## Attributes

| Attribute                                        | Type    | Default                 | Description |
| ------------------------------------------------ | ------- | ----------------------- | ----------- |
| [`backend`](../attributes.md)                    | string  | `""`                    | A [Backend Block](backend.md) reference, defined in [Definitions Block](definitions.md). |
| [`client_id`](../attributes.md)                  | string  | `""`                    | &#9888; Required. The client identifier. |
| [`client_secret`](../attributes.md)              | string  | `""`                    | &#9888; Required. The client password. |
| [`grant_type`](../attributes.md)                 | string  | `""`                    | &#9888; Required. Available values: `client_credentials`. |
| [`retries`](../attributes.md)                    | integer | `1`                     | The number of retries to get the token and resource, if the resource-request responds with `401 Unauthorized` HTTP status code. |
| [`scope`](../attributes.md)                      | string  | `""`                    | A space separated list of requested scopes for the access token. |
| [`token_endpoint`](../attributes.md)             | string  | `""`                    | &#9888; Required. URL of the token endpoint at the authorization server. |
| [`token_endpoint_auth_method`](../attributes.md) | string  | `"client_secret_basic"` | Defines the method to authenticate the client at the token endpoint. |

```diff
! To be able to execute a request the "oauth2" block needs a "backend" block or a "backend" block reference.
```

```diff
! If the "token_endpoint_auth_method" is set to "client_secret_post", the client credentials are transported in the request body. If is set to "client_secret_basic", the client credentials are transported via basic authentication.
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [OAuth2 AC Block](beta-oauth2-ac.md) (Beta)
* &#8674; [OIDC Block](beta-oidc.md) (Beta)
