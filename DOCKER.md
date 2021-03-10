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

### Environment options

| Variable  | Default  | Description  |
|---        |---       |---           |
| COUPER_CONFIG_FILE | `couper.hcl`  | Path to the configuration file. |
| COUPER_LOG_FORMAT | `common`  | Can be set to `json` output which is the _container default_. |
| COUPER_DEFAULT_PORT   | `8080`    | Sets the default port to the given value and does not override explicit `[host:port]` configurations from file. |
| COUPER_XFH    | `false`   | Global configurations which uses the `Forwarded-Host` header instead of the request host.   |
| COUPER_HEALTH_PATH    | `/healthz`   | Path for health-check requests for all servers and ports.   |
| COUPER_NO_PROXY_FROM_ENV | `false` | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| COUPER_REQUEST_ID_FORMAT    | `common`   | If set to `uuid4` a rfc4122 uuid is used for `req.id` and related log fields.   |
| COUPER_ACCESS_LOG_PARENT_FIELD | `""`  | An option for `json` log format to add all log fields as child properties. |
| COUPER_ACCESS_LOG_TYPE_VALUE | `couper_access`  | Value for the log field `type`. |
| COUPER_ACCESS_LOG_REQUEST_HEADERS | `User-Agent, Accept, Referer`  | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_RESPONSE_HEADERS | `Cache-Control, Content-Encoding, Content-Type, Location`  | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_PARENT_FIELD | `""`  | An option for `json` log format to add all log fields as child properties. |
| COUPER_BACKEND_LOG_TYPE_VALUE | `couper_backend`  | Value for the log field `type`. |
| COUPER_BACKEND_LOG_REQUEST_HEADERS | `User-Agent, Accept, Referer`  | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_RESPONSE_HEADERS | `Cache-Control, Content-Encoding, Content-Type, Location`  | A comma separated list of header names whose values should be logged. |
| COUPER_TIMING_IDLE_TIMEOUT | `60s`  | The maximum amount of time to wait for the next request on client connections when keep-alives are enabled. |
| COUPER_TIMING_READ_HEADER_TIMEOUT | `10s`  | The amount of time allowed to read client request headers. |
| COUPER_TIMING_SHUTDOWN_DELAY | `0`  | The amount of time the server is marked as unhealthy until calling server close finally. |
| COUPER_TIMING_SHUTDOWN_TIMEOUT | `0`  | The maximum amount of time allowed to close the server with all running connections. |
