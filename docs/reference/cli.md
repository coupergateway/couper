# Configuration Reference ~ Command Line Interface

Couper is build as binary called `couper` with the following commands:

| Command   | Description |
| --------- | ----------- |
| `help`    | Print the usage for the given command, e.g. `help run`. |
| `run`     | Start the server with given configuration file. |
| `version` | Print the current version and build information. |

```diff
! Note, the options are applied from (lowest to highest priority): settings block, command line interface, environment.
```

**See also:**

* [Environment](environment.md)
* [Settings Block](blocks/settings.md)

## Global Options

| Argument             | Default        | Environment                | Description |
| -------------------- | -------------- | -------------------------- | ----------- |
| `-f`                 | `"couper.hcl"` | `COUPER_FILE`              | File path to the Couper configuration file. |
| `-log-format`        | `"common"`     | `COUPER_LOG_FORMAT`        | Can be set to `json` output format. |
| `-log-pretty`        | `false`        | `COUPER_LOG_PRETTY`        | Option for `json` log format which pretty prints with basic key coloring. |
| `-watch`             | `false`        | `COUPER_WATCH`             | Watch for configuration file changes and reload on modifications. |
| `-watch-retries`     | `5`            | `COUPER_WATCH_RETRIES`     | Maximal retry count for configuration reloads which could not bind the configured port. |
| `-watch-retry-delay` | `"500ms"`      | `COUPER_WATCH_RETRY_DELAY` | Delay duration before next attempt if an error occurs. |

```diff
! The "log-format" and "log-pretty" options can also be configured via settings block or related environment variables.
```

**See also:**

* [Environment](environment.md)
* [Settings Block](blocks/settings.md)

## Run Options

| Argument                         | Default               | Environment                            | Description  |
| -------------------------------- | --------------------- | -------------------------------------- | ------------ |
| `-accept-forwarded-url`          | `""`                  | `COUPER_ACCEPT_FORWARDED_URL`          | Which `X-Forwarded-*` request HTTP header fields should be accepted to change the [Variables](variables/request.md) `request.url`, `request.origin`, `request.protocol`, `request.host`, `request.port`. Comma-separated list of values. Valid values: `proto`, `host`, `port`. |
| `-health-path`                   | `"/healthz"`          | `COUPER_HEALTH_PATH`                   | Path for health-check requests for all servers and ports. |
| `-https-dev-proxy`               | `""`                  | `COUPER_HTTPS_DEV_PROXY`               | List of TLS port mappings to define the TLS listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname. |
| `-no-proxy-from-env`             | `false`               | `COUPER_NO_PROXY_FROM_ENV`             | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| `-p`                             | `8080`                | `COUPER_DEFAULT_PORT`                  | Sets the default port to the given value. Does not override explicit `[host:port]` configurations of the `hosts` attribute in the [Server Blocks](blocks/server.md).          |
| `-request-id-accept-from-header` | `""`                  | `COUPER_REQUEST_ID_ACCEPT_FROM_HEADER` | Name of a client request HTTP header field that transports the [`request.id`](variables/request.md) which Couper takes for logging and transport to the backend (if configured). |
| `-request-id-backend-header`     | `"Couper-Request-ID"` | `COUPER_REQUEST_ID_BACKEND_HEADER`     | Name of a HTTP header field which Couper uses to transport the [`request.id`](variables/request.md) to the backend. |
| `-request-id-client-header`      | `"Couper-Request-ID"` | `COUPER_REQUEST_ID_CLIENT_HEADER`      | Name of a HTTP header field which Couper uses to transport the [`request.id`](variables/request.md) to the client. |
| `-request-id-format`             | `"common"`            | `COUPER_REQUEST_ID_FORMAT`             | If set to `uuid4` a [RFC 4122](https://datatracker.ietf.org/doc/html/rfc4122) UUID is used for [`request.id`](variables/request.md) and related log fields. |
| `-secure-cookies`                | `""`                  | `COUPER_SECURE_COOKIES`                | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |
| `-xfh`                           | `false`               | `COUPER_XFH`                           | Global configuration which uses the `Forwarded-Host` header instead of the request host. |

```diff
! The "run" options can also be configured via settings block or related environment variables.
```

**See also:**

* [Environment](environment.md)
* [Settings Block](blocks/settings.md)

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Blocks](blocks.md)
* &#8674; [Configuration File](config-file.md)
