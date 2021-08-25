# Configuration Reference ~ Environment

```diff
! Note, the options are applied from (lowest to highest priority): settings block, command line interface, environment.
```

**See also:**

* [Command Line Interface](cli.md)
* [Settings Block](blocks/settings.md)

## Options

| Variable                             | Default | Description |
| ------------------------------------ | ------- | ----------- |
| COUPER_FILE                          | `"couper.hcl"` | Path to Couper's configuration file. |
| COUPER_DEFAULT_PORT                  | `8080` | Sets the default port to the given value. Does not override explicit `[host:port]` configurations of the `hosts` attribute in the [Server Blocks](blocks/server.md). |
| COUPER_HEALTH_PATH                   | `"/healthz"` | Path for health-check requests for all servers and ports. |
| COUPER_HTTPS_DEV_PROXY               | `""` | List of TLS port mappings to define the TLS listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname. |
| COUPER_NO_PROXY_FROM_ENV             | `false` | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| COUPER_SECURE_COOKIES                | `""` | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |
| COUPER_WATCH                         | `false` | Set to `true` to watch for configuration file changes. |
| COUPER_WATCH_RETRIES                 | `5` | Maximal retry count for configuration reloads which could not bind the configured port. |
| COUPER_WATCH_RETRY_DELAY             | `"500ms"` | Delay duration before next attempt if an error occurs. |
| COUPER_XFH                           | `false` | Global configurations which uses the `Forwarded-Host` header instead of the request host. |
| **Request ID**                       | | |
| COUPER_REQUEST_ID_ACCEPT_FROM_HEADER | `""` | Name of a client request HTTP header field that transports the [`request.id`](variables/request.md) which Couper takes for logging and transport to the backend (if configured). |
| COUPER_REQUEST_ID_BACKEND_HEADER     | `"Couper-Request-ID"` | Name of a HTTP header field which Couper uses to transport the [`request.id`](variables/request.md) to the backend. |
| COUPER_REQUEST_ID_CLIENT_HEADER      | `"Couper-Request-ID"` | Name of a HTTP header field which Couper uses to transport the [`request.id`](variables/request.md) to the client. |
| COUPER_REQUEST_ID_FORMAT             | `"common"` | If set to `uuid4` a [RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122) UUID is used for [`request.id`](variables/request.md) and related log fields. |
| **Timings**                          | | |
| COUPER_TIMING_IDLE_TIMEOUT           | `"60s"` | The maximum amount of time to wait for the next request on client connections when keep-alives are enabled. |
| COUPER_TIMING_READ_HEADER_TIMEOUT    | `"10s"` | The amount of time allowed to read client request headers. |
| COUPER_TIMING_SHUTDOWN_DELAY         | `0` | The amount of time the server is marked as unhealthy until calling server close finally. |
| COUPER_TIMING_SHUTDOWN_TIMEOUT       | `0` | The maximum amount of time allowed to close the server with all running connections. |
| **Logging**                          | | |
| COUPER_LOG_FORMAT                    | `"common"` | Can be set to `json` output which is the _container default_. |
| COUPER_LOG_PARENT_FIELD              | `""` | An option for `json` log format to add all log fields as child properties. |
| COUPER_LOG_PRETTY                    | `false` | Global option for `json` log format which pretty prints with basic key coloring. |
| COUPER_LOG_TYPE_VALUE                | `"couper_daemon"` | Value for the runtime log field `type`. |
| COUPER_ACCESS_LOG_REQUEST_HEADERS    | `"User-Agent, Accept, Referer"` | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_RESPONSE_HEADERS   | `"Cache-Control, Content-Encoding, Content-Type, Location"` | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_TYPE_VALUE         | `"couper_access"` | Value for the log field `type`. |
| COUPER_BACKEND_LOG_REQUEST_HEADERS   | `"User-Agent, Accept, Referer"` | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_RESPONSE_HEADERS  | `"Cache-Control, Content-Encoding, Content-Type, Location"` | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_TYPE_VALUE        | `"couper_backend"` | Value for the log field `type`. |

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Configuration Types](config-types.md)
* &#8674; [Error Handling](error-handling.md)
