# `request`

The `request` variable contains original information from the incoming client request
and other contextual data.

| Variable                 | Type                            | Description |
| ------------------------ | ------------------------------- | ----------- |
| `body`                   | string                          | Request message body. |
| `context.<label>.<name>` | string                          | Context information from an [Access Control](../access-control.md). See [`request.context`](#requestcontext) section below. |
| `cookies.<name>`         | string                          | Value of the `Cookie` request HTTP header field referenced by `<name>`. |
| `form_body.<name>`       | [list](../config-types.md#list) | Parameter from an `application/x-www-form-urlencoded` body. |
| `headers.<name>`         | string                          | Value of the request HTTP header field referenced by lowercased `<name>`. |
| `host`                   | string                          | Host part of the request URL. |
| `id`                     | string                          | Unique request ID. |
| `json_body.<name>`       | string                          | JSON decoded request message body. Media type must be `application/json` or `application/*+json`. |
| `method`                 | string                          | HTTP request method. |
| `origin`                 | string                          | Origin part of the request URL. |
| `path`                   | string                          | Path part of the request URL. |
| `path_params.<name>`     | string                          | Value of a named path parameter. See [`path-parameter`](#requestcontext) section below. |
| `port`                   | integer                         | Port part of the request URL. |
| `protocol`               | string                          | Request protocol. |
| `query.<name>`           | [list](../config-types.md#list) | Query parameter values from the request URL referenced by `<name>`. |
| `url`                    | string                          | Request URL. |

## Path Parameter

An [Endpoint Block](../blocks/endpoint.md) `label` could be defined as `"/images/{vehicle}/{color}/view"`
to be able to access the named path parameter `vehicle` and `color` via `request.path_params.*`.

The values would map as following for the request path `/images/bus/yellow/view`:

| Variable                      | Value    |
| ----------------------------- | -------- |
| `request.path_params.vehicle` | `bus`    |
| `request.path_params.color`   | `yellow` |

## `request.context`

The value of `context.<label>` depends on the type of [Access Control](../access-control.md)
block referenced by `<label>`.

For a [JWT Block](../blocks/jwt.md) the variable contains claims from the JWT used
for [Access Control](../access-control.md).

For an [OAuth2 AC Block](../blocks/beta-oauth2-ac.md) (Beta) the variable contains
the response from the token endpoint, e.g.

* `access_token`: the access token retrieved from the token endpoint.
* `token_type`: the token type.
* `expires_in`: the token lifetime.
* `scope`: the granted scope (if different from the requested scope).

and for an [OIDC Block](../blocks/beta-oidc.md) (Beta) the variable contains additionally:

* `id_token`: the ID token.
* `id_token_claims`: a map of claims from the ID token.
* `userinfo`: a map of claims retrieved from the userinfo endpoint.

For a [SAML Block](../blocks/saml.md) the variable contains

* `sub`: the `NameID` of the SAML assertion's `Subject`.
* `exp`: optional session expiration date (value of `SessionNotOnOrAfter` of the SAML assertion).
* `attributes`: a map of attributes from the SAML assertion.

```diff
! Same context data is available on "backend_requests.<label>.context.<label>" and "backend_responses.<label>.context.<label>".
```

**See also:**

* [`backend_requests`](backend-requests.md)
* [`backend_responses`](backend-responses.md)

## Examples

* [Variables and Expressions](../examples.md#variables-and-expressions)

-----

## Navigation

* &#8673; [Variables](../variables.md)
* &#8672; [`env`](env.md)
* &#8674; [`backend_requests`](backend-requests.md)
