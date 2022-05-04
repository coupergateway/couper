# Errors

- [Errors](#errors)
  - [Introduction](#introduction)
  - [Error messages](#error-messages)
  - [Access control error_handler](#access-control-error_handler)
  - [Permissions related error_handler](#permissions-related-error_handler)
  - [Endpoint related error_handler](#endpoint-related-error_handler)
  - [Error types](#error-types)
    - [Access control error types](#access-control-error-types)
    - [API error types](#api-error-types)
    - [Endpoint error types](#endpoint-error-types)

## Introduction

Errors can occur in various places: due to invalid client requests or problems on the backend and network side.
Couper specifies some generic error categories (like `configuration`, `server`, `backend` or `access_control`) to help you identify the occurring problems faster.

## Error messages

Error messages are only sent to the client as a summary.
Detailed information is provided via log message. This way, all information can be viewed without accidentally revealing confidential information.

## Access control `error_handler`

Access control errors in particular require special handling, e.g. sending a specific response for missing login credentials.
For this purpose every access control definition of `basic_auth`, `jwt`, `oidc` or `saml2` can define one or multiple [`error_handler` blocks](REFERENCE.md#error-handler-block) with one or more defined error type labels listed below.

## Permissions related `error_handler`

Required permissions are configured for `api` or `endpoint` blocks. So errors caused by insufficient permissions can be handled at API or endpoint levels.

## Endpoint related `error_handler`

A [sequence](REFERENCE.md#endpoint-sequence), a simple `request` or `proxy` error can be handled in combination with the `expected_status` attribute for `request`
and `proxy` block definitions and an [`error_handler` block](REFERENCE.md#error-handler-block) with the [related](#endpoint-error-types) type label.

## Error types

All errors have a specific type. You can find it in the log field `error_type`. Furthermore, errors can be associated with a list of less specific types. Your error handlers will be evaluated from the most to the least specific one. Only the first matching error handler is executed.

### Access control error types

| Type (and super types)                          | Description                                                                                                                  | Default handling                                                            |
|:------------------------------------------------|:-----------------------------------------------------------------------------------------------------------------------------|:----------------------------------------------------------------------------|
| `basic_auth`                                    | All `basic_auth` related errors, e.g. unknown user or wrong password.                                                        | Send error template with status `401` and `WWW-Authenticate: Basic` header. |
| `basic_auth_credentials_missing` (`basic_auth`) | Client does not provide any credentials.                                                                                     | Send error template with status `401` and `WWW-Authenticate: Basic` header. |
| `jwt`                                           | All `jwt` related errors.                                                                                                    | Send error template with status `403`.                                      |
| `jwt_token_missing` (`jwt`)                     | No token provided with configured token source.                                                                              | Send error template with status `401`.                                      |
| `jwt_token_expired` (`jwt`)                     | Given token is valid but expired.                                                                                            | Send error template with status `403`.                                      |
| `jwt_token_invalid` (`jwt`)                     | The token is syntactically not a JWT, or not sufficient, e.g. because required claims are missing or have unexpected values. | Send error template with status `403`.                                      |
| `saml` (or `saml2`)                             | All `saml` related errors                                                                                                    | Send error template with status `403`.                                      |
| `oauth2`                                        | All `beta_oauth2`/`oidc` related errors                                                                                      | Send error template with status `403`.                                      |

### API error types

| Type (and super types)                          | Description                                                                                             | Default handling                                                                                             |
|:------------------------------------------------|:--------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------|
| `backend`                                       | All catchable `backend` related errors                                                                  | Send error template with status `502`.                                                                       |
| `backend_openapi_validation` (`backend`)        | Backend request or response is invalid                                                                  | Send error template with status code `400` for invalid backend request or `502` for invalid backend response. |
| `backend_timeout` (`backend`)                   | A backend request timed out                                                                             | Send error template with status `504`.                                                                       |
| `beta_insufficient_permissions`                 | The permission required for the requested operation is not in the permissions granted to the requester. | Send error template with status `403`.                                                                       |

### Endpoint error types

| Type (and super types)                          | Description                                                                                             | Default handling                                                                                             |
|:------------------------------------------------|:--------------------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------|
| `backend`                                       | All catchable `backend` related errors                                                                  | Send error template with status `502`.                                                                       |
| `backend_openapi_validation` (`backend`)        | Backend request or response is invalid                                                                  | Send error template with status code `400` for invalid backend request or `502` for invalid backend response. |
| `backend_timeout` (`backend`)                   | A backend request timed out                                                                             | Send error template with status `504`.                                                                       |
| `beta_insufficient_permissions`                 | The permission required for the requested operation is not in the permissions granted to the requester. | Send error template with status `403`.                                                                       |
| `sequence`                                      | A `request` or `proxy` block request has been failed while depending on another one                     | Send error template with status `502`.                                                                       |
| `unexpected_status`                             | A `request` or `proxy` block response status code does not match the to `expected_status` list          | Send error template with status `502`.                                                                       |
