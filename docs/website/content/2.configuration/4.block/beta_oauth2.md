# OAuth2 AC (Beta)

The `beta_oauth2` block lets you configure the [`oauth2_authorization_url()` function](/configuration/functions) and an access
control for an OAuth2 **Authorization Code Grant Flow** redirect endpoint.
Like all [access control](/configuration/access-control) types, the `beta_oauth2` block is defined in the [`definitions` block](/configuration/block/definitions) and can be referenced in all configuration blocks by its required _label_.

| Block name    | Context                                 | Label            | Nested block(s)                                                                                                  |
|:--------------|:----------------------------------------|:-----------------|:-----------------------------------------------------------------------------------------------------------------|
| `beta_oauth2` | [Definitions Block](/configuration/block/definitions)        | &#9888; required | [Backend Block](/configuration/block/backend), [Error Handler Block](/configuration/block/error_handler), [JWT Signing Profile Block](jwt_signing_profile) |

A nested `jwt_signing_profile` block is used to create a client assertion if `token_endpoint_auth_method` is either `"client_secret_jwt"` or `"private_key_jwt"`.

::attributes
---
values: [
  {
    "default": "",
    "description": "The authorization server endpoint URL used for authorization.",
    "name": "authorization_endpoint",
    "type": "string"
  },
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for token requests.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client identifier.",
    "name": "client_id",
    "type": "string"
  },
  {
    "default": "",
    "description": "The client password.",
    "name": "client_secret",
    "type": "string"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "The grant type. Required, to be set to: `\"authorization_code\"`",
    "name": "grant_type",
    "type": "string"
  },
  {
    "default": "",
    "description": "The Couper endpoint for receiving the authorization code. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the [`accept_forwarded_url` attribute](settings) if Couper is running behind a proxy.",
    "name": "redirect_uri",
    "type": "string"
  },
  {
    "default": "",
    "description": "A space separated list of requested scope values for the access token.",
    "name": "scope",
    "type": "string"
  },
  {
    "default": "",
    "description": "The authorization server endpoint URL used for requesting the token.",
    "name": "token_endpoint",
    "type": "string"
  },
  {
    "default": "\"client_secret_basic\"",
    "description": "Defines the method to authenticate the client at the token endpoint. If set to `\"client_secret_post\"`, the client credentials are transported in the request body. If set to `\"client_secret_basic\"`, the client credentials are transported via Basic Authentication. If set to `\"client_secret_jwt\"`, the client is authenticated via a JWT signed with the `client_secret`. If set to `\"private_key_jwt\"`, the client is authenticated via a JWT signed with its private key (see `jwt_signing_profile` block).",
    "name": "token_endpoint_auth_method",
    "type": "string"
  },
  {
    "default": "",
    "description": "The method to verify the integrity of the authorization code flow. Available values: `\"ccm_s256\"` (`code_challenge` parameter with `code_challenge_method` `S256`), `\"state\"` (`state` parameter)",
    "name": "verifier_method",
    "type": "string"
  },
  {
    "default": "",
    "description": "The value of the (unhashed) verifier. E.g. using cookie value created with `oauth2_verifier()` function](../functions)",
    "name": "verifier_value",
    "type": "string"
  }
]

---
::

If the authorization server supports the `code_challenge_method` `S256` (a.k.a. PKCE, see RFC 7636), we recommend `verifier_method = "ccm_s256"`.

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](/configuration/modifiers#request-header) in a [backend block](/configuration/block/backend).

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for token requests.",
    "name": "backend"
  },
  {
    "description": "Configures a [JWT signing profile](/configuration/block/jwt_signing_profile) to create a client assertion if `token_endpoint_auth_method` is either `\"client_secret_jwt\"` or `\"private_key_jwt\"`.",
    "name": "jwt_signing_profile"
  }
]

---
::
