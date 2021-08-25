# Backend Block

The `backend` block configures the connection to a local or remote backend service.

| Block name | Label               | Related blocks |
| ---------- | ------------------- | -------------- |
| `backend`  | &#10003; (optional) | [Definitions Block](definitions.md), [OAuth2 AC Block](beta-oauth2-ac.md) (Beta), [OAuth2 CC Block](oauth2-cc.md), [OIDC Block](beta-oidc.md) (Beta), [Proxy Block](proxy.md), [Request Block](request.md) |

```diff
! Backends can be defined in the "definitions" block and referenced (reused) by "label".
```

```diff
! The "label" is required if the "backend" block is defined in the "definitions" block.
```

## Nested blocks

* [OAuth2 CC Block](oauth2-cc.md)
* [OpenAPI Block](openapi.md)

## Attributes

| Attribute                                            | Type                                    | Default   | Description |
| ---------------------------------------------------- | --------------------------------------- | --------- | ----------- |
| [`basic_auth`](../attributes.md)                     | string                                  | `""`      | Basic auth for the upstream request in format `username:password`. |
| [`connect_timeout`](../attributes.md)                | [duration](../config-types.md#duration) | `"10s"`   | The total timeout for dialing and connect to the origin. |
| [`disable_certificate_validation`](../attributes.md) | bool                                    | `false`   | Disables the peer certificate validation. |
| [`disable_connection_reuse`](../attributes.md)       | bool                                    | `false`   | Disables reusage of connections to the origin. |
| [`hostname`](../attributes.md)                       | string                                  | `""`      | Value of the `Host` HTTP header field for the origin request. Since `hostname` replaces the request `Host` HTTP header field, the value will also be used for a server identity check during a TLS handshake with the origin. |
| [`http2`](../attributes.md)                          | bool                                    | `false`   | Enables the HTTP2 support. |
| [`max_connections`](../attributes.md)                | integer                                 | &#10005;  | The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. |
| [`origin`](../attributes.md)                         | string                                  | `""`      | &#9888; Required. URL to connect for backend requests. Must start with a scheme, e.g. `http://...`. |
| [`path`](../attributes.md)                           | string                                  | `""`      | Changeable part of the upstream URL. Changes the path suffix of the outgoing request URL. |
| [`path_prefix`](../attributes.md)                    | string                                  | `""`      | Prefixes all backend request paths with the given prefix. |
| [`proxy`](../attributes.md)                          | string                                  | `""`      | A proxy URL for the related origin request. |
| [`timeout`](../attributes.md)                        | [duration](../config-types.md#duration) | `"300s"`  | The total deadline duration a backend request has for write and read/pipe. |
| [`ttfb_timeout`](../attributes.md)                   | [duration](../config-types.md#duration) | `"60s"`   | The duration from writing the full request to the origin and receiving the answer. |
| Modifiers                                            | &#10005;                                | &#10005;  | The `backend` block supports [Modifiers](../modifiers.md). |
| Parameters                                           | &#10005;                                | &#10005;  | The `backend` block supports [Parameters](../parameters.md). |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [API Block](api.md)
* &#8674; [Basic Auth Block](basic-auth.md)
