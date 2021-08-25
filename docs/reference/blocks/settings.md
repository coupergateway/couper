# Settings Block

The `settings` block configures the more basic and global behavior of the Couper
gateway instance.

| Block name | Label    | Related blocks |
| ---------- | :------: | :------------: |
| `settings` | &#10005; | &#10005;       |

## Attributes

| Attribute                                           | Type                            | Default               | Description |
| --------------------------------------------------- | ------------------------------- | --------------------- | ----------- |
| [`accept_forwarded_url`](../attributes.md)          | [list](../config-types.md#list) | `[]`                  | Which `X-Forwarded-*` request HTTP header fields should be accepted to change the [Variables](../variables/request.md) `request.url`, `request.origin`, `request.protocol`, `request.host`, `request.port`. Comma-separated list of values. Valid values: `proto`, `host`, `port`. |
| [`default_port`](../attributes.md)                  | integer                         | `8080`                | Sets the default port to the given value. Does not override explicit `[host:port]` configurations of the `hosts` attribute in the [Server Blocks](server.md). |
| [`health_path`](../attributes.md)                   | string                          | `"/healthz"`          | Path for health-check requests for all servers and ports. |
| [`https_dev_proxy`](../attributes.md)               | [list](../config-types.md#list) | `[]`                  | List of TLS port mappings to define the TLS listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname. |
| [`log_format`](../attributes.md)                    | string                          | `"common"`            | Can be set to `json` output format. |
| [`log_pretty`](../attributes.md)                    | bool                            | `false`               | Option for `json` log format which pretty prints with basic key coloring. |
| [`no_proxy_from_env`](../attributes.md)             | bool                            | `false`               | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| [`request_id_accept_from_header`](../attributes.md) | string                          | `""`                  | Name of a client request HTTP header field that transports the [`request.id`](../variables/request.md) which Couper takes for logging and transport to the backend (if configured). |
| [`request_id_backend_header`](../attributes.md)     | string                          | `"Couper-Request-ID"` | Name of a HTTP header field which Couper uses to transport the [`request.id`](../variables/request.md) to the backend. |
| [`request_id_client_header`](../attributes.md)      | string                          | `"Couper-Request-ID"` | Name of a HTTP header field which Couper uses to transport the [`request.id`](../variables/request.md) to the client. |
| [`request_id_format`](../attributes.md)             | string                          | `"common"`            | If set to `uuid4` a [RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122) UUID is used for [`request.id`](../variables/request.md) and related log fields. |
| [`secure_cookies`](../attributes.md)                | string                          | `""`                  | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |
| [`xfh`](../attributes.md)                           | bool                            | `false`               | Global configuration which uses the `Forwarded-Host` header instead of the request host. |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Server Block](server.md)
* &#8674; [SPA Block](spa.md)
