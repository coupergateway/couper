# Configuration Reference ~ Blocks

In contrast to [Attributes](attributes.md), Blocks in Couper configuration never
have an [Equals (`=`)](https://en.wikipedia.org/wiki/Equals_sign) sign before the
block surrounded by curly braces:

```hcl
// Blocks:
block {...}
block_labled "label" {...}

// Attributes:
attribute = "value"
attribute_map = {...}
```

## Blocks

Click on the block name to see more details.

| Block                                                      | Description |
| ---------------------------------------------------------- | ----------- |
| [API Block](blocks/api.md)                                 | The `api` block bundles [Endpoint Blocks](blocks/endpoint.md) under a certain `base_path`. |
| [Backend Block](blocks/backend.md)                         | The `backend` block configures the connection to a local or remote backend service. |
| [Basic Auth Block](blocks/basic-auth.md)                   | The `basic_auth` block configures the basic auth. |
| [CORS Block](blocks/cors.md)                               | The `cors` block configures the CORS behavior in Couper. |
| [Defaults Block](blocks/defaults.md)                       | The `defaults` block configures default values for the Couper configuration. |
| [Definitions Block](blocks/definitions.md)                 | The `definitions` block contains reusable (e.g. [Backend Block](blocks/backend.md)) or [Access Control](access-control.md) configurations. |
| [Endpoint Block](blocks/endpoint.md)                       | An `endpoint` block defines an entry point of Couper. |
| [Error Handler Block](blocks/error-handler.md)             | The Error Handler `label` specifies which [Error Type](error-handling.md#error-types) should be handled. |
| [Files Block](blocks/files.md)                             | The `files` block configures the file serving. |
| [JWT Block](blocks/jwt.md)                                 | The `jwt` block configures the JSON Web Token access control. |
| [JWT Signing Profile Block](blocks/jwt-signing-profile.md) | The `jwt_signing_profile` block configures a JSON Web Token signing profile. |
| [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta)         | The `beta_oauth2` block configures the [`beta_oauth_authorization_url()` function](functions/beta-oauth-authorization-url.md) and an [Access Control](access-control.md) for an OAuth2 Authorization Code Grant Flow redirect endpoint. |
| [OAuth2 CC Block](blocks/oauth2-cc.md)                     | The `oauth2` block in the [Backend Block](blocks/backend.md) context configures the OAuth2 Client Credentials Flow to request a bearer token for the backend request. |
| [OIDC Block](blocks/beta-oidc.md) (Beta)                   | The `beta_oidc` block configures the [`beta_oauth_authorization_url()` function](functions/beta-oauth-authorization-url.md) and an [Access Control](access-control.md) for an OIDC Authorization Code Grant Flow redirect endpoint. |
| [OpenAPI Block](blocks/openapi.md)                         | The `openapi` block configures the backends proxy behavior to validate outgoing and incoming requests to and from the origin. |
| [Proxy Block](blocks/proxy.md)                             | The `proxy` block creates and executes a proxy request to a backend service. |
| [Request Block](blocks/request.md)                         | The `request` block creates and executes a request to a backend service. |
| [Response Block](blocks/response.md)                       | The `response` block creates and sends a client response. |
| [SAML Block](blocks/saml.md)                               | The `saml` block configures the [`saml_sso_url()` function](functions/saml-sso-url.md) and an [Access Control](access-control.md) for a SAML Assertion Consumer Service (ACS) endpoint. |
| [Server Block](blocks/server.md)                           | The `server` block is one of the root configuration blocks of Couper's configuration file. |
| [Settings Block](blocks/settings.md)                       | The `settings` block configures the more basic and global behavior of the Couper gateway instance. |
| [SPA Block](blocks/spa.md)                                 | The `spa` block configures the Web serving for SPA assets. |
| [Websockets Block](blocks/websockets.md)                   | The `websockets` block activates support for websocket connections in Couper. |

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Beta Features](beta-features.md)
* &#8674; [Command Line Interface](cli.md)
