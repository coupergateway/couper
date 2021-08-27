# Configuration Reference ~ Error Handling

Errors can occur in various places: due to invalid client requests or problems on
the backend and network side. Couper specifies some generic error categories (like
`configuration`, `server`, `backend` or `access_control`) to help You identify the
occurring problems faster.

## Error Messages

Error messages are only sent to the client as a summary. Detailed information is
provided via log messages. This way, all information can be viewed without accidentally
revealing confidential information.

## Access Control `error_handler`

[Access Control](access-control.md) errors in particular require special handling,
e.g. sending a specific response for missing login credentials. For this purpose
every access control definition of [Basic Auth Block](blocks/basic-auth.md),
[JWT Block](blocks/jwt.md), [OAuth2 AC Block](blocks/beta-oauth2-ac.md) (Beta),
[OIDC Block](blocks/beta-oidc.md) (Beta) or [SAML Block](blocks/saml.md) can define
one or multiple [Error Handler Blocks](blocks/error-handler.md) with one or more
defined [Error Type](#error-types) labels listed below.

### Error Types

All errors have a specific type. You can find it in the log field `error_type`.
Furthermore, errors can be associated with a list of less specific types. Your error
handlers will be evaluated from the most to the least specific one. Only the first
matching error handler is executed.

| Type (super type)                               | Description | Default handling |
| ----------------------------------------------- | ----------- | ---------------- |
| `basic_auth`                                    | All `basic_auth` related errors, e.g. unknown user or wrong password. | Send error template with HTTP status code `401` and `WWW-Authenticate: Basic` HTTP header field. |
| `basic_auth_credentials_missing` (`basic_auth`) | Client does not provide any credentials. | Send error template with HTTP status code `401` and `WWW-Authenticate: Basic` HTTP header field. |
| `jwt`                                           | All `jwt` related errors. | Send error template with HTTP status code `403`. |
| `jwt_token_missing` (`jwt`)                     | No token provided with configured token source. | Send error template with HTTP status code `401`. |
| `jwt_token_expired` (`jwt`)                     | Given token is valid but expired. | Send error template with HTTP status code `403`. |
| `jwt_token_invalid` (`jwt`)                     | The token is not sufficient, e.g. because required claims are missing or have unexpected values. | Send error template with HTTP status code `403`. |
| `oauth2` (Beta)                                 | All `beta_oauth2`/`beta_oidc` related errors. | Send error template with HTTP status code `403`. |
| `saml2`                                         | All `saml2` related errors. | Send error template with HTTP status code `403`. |

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Environment](environment.md)
* &#8674; [Examples](examples.md)
