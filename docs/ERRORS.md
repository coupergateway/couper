# Errors

* [Errors](#errors)
    * [Introduction](#introduction)
    * [Messages](#messages)
    * [Access-Control error_handler](#access-control-error_handler)
        * [error_handler specification](#error_handler-specification)
        * [Error-Types](#error-types)

## Introduction

Errors should be handled and could occur due to user input or issues on backend and network side.
There are some generic categories like `configuration`, `server`, `backend` or `access_control`.
Those categories will help you to identify the related issues faster.

## Messages

Couper errors distinguish between the synopsis which will be sent to the client, and
the log message with more error context to prevent accidentally exposing confidential data.

## Access-Control `error_handler`

Especially some access-control errors are expected to occur and may need special handling,
e.g. sending a specific response for missing login credentials.
To do so every access-control definition of `basic_auth`, `jwt` or `saml2` can define one or multiple
`error_handler` with one or more defined error-type labels listed below.

### `error_handler` specification

This handler behaves like an [endpoint](README.md#endpoint-block) and almost the same attributes are available.

Except the following *endpoint* attributes are not available for `error_handler`: 
* `access_control`
* `disable_access_control`
* `request_body_limit`

Also, the [modifier](README.md#modifier) and [query params](README.md#query-parameter) can be configured.
The label can be omitted to catch all errors which are related to this access control definition.
To react for a specific [error-type](#error-types) list them per label.

```hcl
error_handler "error-type" "additional-type" {
  error_file = "my_custom_file.html"
  response {}
  request {}
  proxy {}
}
```

### Error-Types

| Type                              | Description                                           | Default handling |
|:----------------------------------|:------------------------------------------------------|:-----------------|
| `basic_auth`                      | All `basic_auth` related errors, e.g. unknown user or wrong password. | Send error template with status `401` and `Www-Authenticate: Basic` header. |
| `basic_auth_credentials_required` | Client does not provide any credentials. | Send error template with status `401` and `Www-Authenticate: Basic` header. |
| `jwt`                             | All `jwt` related errors. | Send error template with status `403`. |
| `jwt_token_required`              | No token provided with configured token source.  | Send error template with status `401`. |
| `jwt_token_expired`               | Given token is valid but expired. | Send error template with status `403`. |
| `jwt_claims`                      | Claim related errors like missing claims or unexpected values. | Send error template with status `403`. |
| `saml2`                           | All `saml2` related errors | Send error template with status `403`. |
