# Backend

The `backend` block defines the connection to a local/remote backend service.

&#9888; Backends can be defined in the [Definitions Block](#definitions-block) and referenced by _label_.

| Block name | Context                                                                                                                                                                                                                                   | Label                                                                     | Nested block(s)                                                                                  |
|:-----------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| `backend`  | [Definitions Block](#definitions-block), [Proxy Block](#proxy-block), [Request Block](#request-block), [OAuth2 CC Block](#oauth2-block), [JWT Block](#jwt-block), [OAuth2 AC Block (beta)](#beta-oauth2-block), [OIDC Block](#oidc-block) | &#9888; required, when defined in [Definitions Block](#definitions-block) | [OpenAPI Block](#openapi-block), [OAuth2 CC Block](#oauth2-block), [Health Block](#health-block) |

| Attribute(s)                     | Type                  | Default         | Description                                                                                   | Characteristic(s)                                                                                                                          | Example                           |
|:---------------------------------|:----------------------|:----------------|:----------------------------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------|:----------------------------------|
| `basic_auth`                     | string                | -               | Basic auth for the upstream request.                                                          | format is `"username:password"`                                                                                                            | -                                 |
| `custom_log_fields`              | object                | -               | Defines log fields for [Custom Logging](LOGS.md#custom-logging).                              | -                                                                                                                                          | -                                 |
| `hostname`                       | string                | -               | Value of the HTTP host header field for the origin request.                                   | Since `hostname` replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin. | -                                 |
| `origin`                         | string                | -               | URL to connect to for backend requests.                                                       | &#9888; required.  &#9888; Must start with one of the URI schemes `https` or `http`.                                                       | -                                 |
| `path`                           | string                | -               | Changeable part of upstream URL.                                                              | -                                                                                                                                          | -                                 |
| `path_prefix`                    | string                | -               | Prefixes all backend request paths with the given prefix                                      | -                                                                                                                                          | -                                 |
| `connect_timeout`                | [duration](#duration) | `"10s"`         | The total timeout for dialing and connect to the origin.                                      | -                                                                                                                                          | -                                 |
| `disable_certificate_validation` | bool                  | `false`         | Disables the peer certificate validation.                                                     | -                                                                                                                                          | -                                 |
| `disable_connection_reuse`       | bool                  | `false`         | Disables reusage of connections to the origin.                                                | -                                                                                                                                          | -                                 |
| `http2`                          | bool                  | `false`         | Enables the HTTP2 support.                                                                    | -                                                                                                                                          | -                                 |
| `max_connections`                | integer               | `0` (unlimited) | The maximum number of concurrent connections in any state (_active_ or _idle_) to the origin. | -                                                                                                                                          | -                                 |
| `proxy`                          | string                | -               | A proxy URL for the related origin request.                                                   | -                                                                                                                                          | `"http://SERVER-IP_OR_NAME:PORT"` |
| `timeout`                        | [duration](#duration) | `"300s"`        | The total deadline duration a backend request has for write and read/pipe.                    | -                                                                                                                                          | -                                 |
| `ttfb_timeout`                   | [duration](#duration) | `"60s"`         | The duration from writing the full request to the origin and receiving the answer.            | -                                                                                                                                          | -                                 |
| `use_when_unhealthy`             | bool                  | `false`         | Ignores the [health](#health-block) state and continues with the outgoing request             | -                                                                                                                                          | -                                 |
| [Modifiers](#modifiers)          | -                     | -               | All [Modifiers](#modifiers)                                                                   | -                                                                                                                                          | -                                 |

#### Duration

| Duration units | Description  |
|:---------------|:-------------|
| `ns`           | nanoseconds  |
| `us` (or `µs`) | microseconds |
| `ms`           | milliseconds |
| `s`            | seconds      |
| `m`            | minutes      |
| `h`            | hours        |