# Errors

* [Errors](#errors)
    * [Introduction](#introduction)
    * [Messages](#messages)
    * [Access-Control error_handler](#access-control-error_handler)
        * [Error-Types](#error-types)

## Introduction

Errors should be handled and could occur due to user input or issues on backend and network side.
There are some generic categories like `configuration`, `server`, `backend` or `access_control`.
Those categories will help you to identify the related issues faster.

## Messages

Couper errors distinguish between the synopsis which will be sent to the client, and
the log message with more error context to prevent accidentally exposing confidential data.

## Access-Control `error_handler`

Especially some access-control errors are expected ones and may be handled on a specific manner.
To do so every access-control definition of `basic_auth`, `jwt` or `saml2` can define one or multiple
`error_handler` with a defined error-type label listed below.

Syntax:

```hcl
# Without label will catch all validation errors related to this access control. 
error_handler "error-type-a" "error-type-b" {
  # Behaves like an endpoint
  response {
    status = 403
  }
}
```

### Error-Types

| Type                              | Description |
|:----------------------------------|:------------|
| `basic_auth`                      | All `basic_auth` related errors |
| `basic_auth_credentials_required` | Client does not provide any credentials |
| `jwt`                             | All `jwt` related errors |
| `jwt_token_required`              | No token provided with configured token source  |
| `jwt_token_expired`               | Given token is valid but expired |
| `jwt_claims`                      | Claim related errors like missing or unexpected value |
| `saml2`                           | All `saml2` related errors |
