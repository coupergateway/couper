# Errors

- [Errors](#errors)
  - [Introduction](#introduction)
  - [Error messages](#error-messages)
  - [Access control error_handler](#access-control-error_handler)
    - [error_handler specification](#error_handler-specification)
    - [Error types](#error-types)

## Introduction

Errors can occur in various places: due to invalid client requests or problems on the backend and network side.
Couper specifies some generic error categories (like `configuration`, `server`, `backend` or `access_control`) to help you identify the occurring problems faster.

## Error messages

Error messages are only sent to the client as a summary.
Detailed information is provided via log message. This way, all information can be viewed without accidentally revealing confidential information.

## Access control `error_handler`

Access control errors in particular require special handling, e.g. sending a specific response for missing login credentials.
For this purpose every access control definition of `basic_auth`, `jwt` or `saml2` can define one or multiple `error_handler` with one or more defined error type labels listed below.

### `error_handler` specification

The error handler label specifies which [error type](#error-types)
should be handled. Multiple labels are allowed. The label can be omitted to catch all errors which are related to this access control definition. This has the same behavior as the error type `*`, that catches all errors explicitly.

This handler behaves like an [endpoint](REFERENCE.md#endpoint-block). It can have the same attributes **except** the following:

- `access_control`
- `disable_access_control`
- `request_body_limit`

Example:

```hcl
error_handler "jwt_token_missing" {
  error_file = "my_custom_file.html"
  response {}
}
```

### Error types

All errors have a specific type. You can find it in the log field `error_type`. Furthermore, errors can be associated with a list of less specific types. Your error handlers will be evaluated from the most to the least specific one. Only the first matching error handler is executed.

| Type (and super types)                          | Description                                                                                      | Default handling                                                            |
| :---------------------------------------------- | :----------------------------------------------------------------------------------------------- | :-------------------------------------------------------------------------- |
| `basic_auth`                                    | All `basic_auth` related errors, e.g. unknown user or wrong password.                            | Send error template with status `401` and `WWW-Authenticate: Basic` header. |
| `basic_auth_credentials_missing` (`basic_auth`) | Client does not provide any credentials.                                                         | Send error template with status `401` and `WWW-Authenticate: Basic` header. |
| `jwt`                                           | All `jwt` related errors.                                                                        | Send error template with status `403`.                                      |
| `jwt_token_missing` (`jwt`)                     | No token provided with configured token source.                                                  | Send error template with status `401`.                                      |
| `jwt_token_expired` (`jwt`)                     | Given token is valid but expired.                                                                | Send error template with status `403`.                                      |
| `jwt_token_invalid` (`jwt`)                     | The token is not sufficient, e.g. because required claims are missing or have unexpected values. | Send error template with status `403`.                                      |
| `saml2`                                         | All `saml2` related errors                                                                       | Send error template with status `403`.                                      |
| `oauth2`                                        | All `beta_oauth2`/`beta_oidc` related errors                                                     | Send error template with status `403`.                                      |
