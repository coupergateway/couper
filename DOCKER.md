# Couper

Couper is designed to support developers building and operating API-driven Web projects by offering security and observability functionality in a frontend gateway component.

_For additional information, tutorials and documentation please visit the [couper repository](https://github.com/avenga/couper)._

## Usage

Couper requires a [configuration file](https://github.com/avenga/couper/tree/master/docs#conf_file) which have to be provided on start.
See our [documentation](https://github.com/avenga/couper/tree/master/docs) how to configure _couper_.

This image contains a basic configuration to serve files from `/htdocs` directory.

```bash
docker run --rm -p 8080:8080 -v `pwd`:/htdocs avenga/couper
```

## Command

The entrypoint of the image is the `/couper` binary. The command is `run`.

Therefore `docker run avenga/couper` runs `/couper run`. You could also use other commands directly:

```shell
$ docker run avenga/couper version
$ docker run avenga/couper run -watch -p 8081
```

### Environment options

| Variable  | Default  | Description  |
|---        |---       |---           |
| COUPER_FILE | `couper.hcl`  | Path to the configuration file. |
| COUPER_WATCH | `false`  | Set to `true` to watch for configuration file changes. |
| COUPER_WATCH_RETRY_DELAY | `500ms`  | Delay duration before next attempt if an error occurs. |
| COUPER_WATCH_RETRIES | `5`  | Maximal retry count for configuration reloads which could not bind the configured port. |
| COUPER_DEFAULT_PORT   | `8080`    | Sets the default port to the given value and does not override explicit `[host:port]` configurations from file. |
| COUPER_XFH    | `false`   | Global configurations which uses the `Forwarded-Host` header instead of the request host.   |
| COUPER_HEALTH_PATH    | `/healthz`   | Path for health-check requests for all servers and ports.   |
| COUPER_NO_PROXY_FROM_ENV | `false` | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| COUPER_REQUEST_ID_FORMAT    | `common`   | If set to `uuid4` a rfc4122 uuid is used for `request.id` and related log fields.   |
| COUPER_SECURE_COOKIES | `""` | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |
| COUPER_TIMING_IDLE_TIMEOUT | `60s`  | The maximum amount of time to wait for the next request on client connections when keep-alives are enabled. |
| COUPER_TIMING_READ_HEADER_TIMEOUT | `10s`  | The amount of time allowed to read client request headers. |
| COUPER_TIMING_SHUTDOWN_DELAY | `0`  | The amount of time the server is marked as unhealthy until calling server close finally. |
| COUPER_TIMING_SHUTDOWN_TIMEOUT | `0`  | The maximum amount of time allowed to close the server with all running connections. |
|   |   |   |
| COUPER_LOG_FORMAT | `common`  | Can be set to `json` output which is the _container default_. |
| COUPER_LOG_PRETTY | `false`  | Global option for `json` log format which pretty prints with basic key coloring. |
| COUPER_LOG_PARENT_FIELD | `""`  | An option for `json` log format to add all log fields as child properties. |
| COUPER_LOG_TYPE_VALUE | `couper_daemon`  | Value for the runtime log field `type`. |
| COUPER_ACCESS_LOG_TYPE_VALUE | `couper_access`  | Value for the log field `type`. |
| COUPER_ACCESS_LOG_REQUEST_HEADERS | `User-Agent, Accept, Referer`  | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_RESPONSE_HEADERS | `Cache-Control, Content-Encoding, Content-Type, Location`  | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_TYPE_VALUE | `couper_backend`  | Value for the log field `type`. |
| COUPER_BACKEND_LOG_REQUEST_HEADERS | `User-Agent, Accept, Referer`  | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_RESPONSE_HEADERS | `Cache-Control, Content-Encoding, Content-Type, Location`  | A comma separated list of header names whose values should be logged. |
