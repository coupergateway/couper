# Couper

![Couper](https://raw.githubusercontent.com/avenga/couper/master/docs/website/public/img/couper-logo.svg)

Couper is designed to support developers building and operating API-driven Web projects by offering security and observability functionality in a frontend gateway component.

_For additional information, tutorials and documentation please visit the [Couper repository](https://github.com/avenga/couper)._

## Usage

Couper requires a [configuration file](https://docs.couper.io/configuration/configuration-file) which have to be provided on start.
See our [documentation](https://docs.couper.io/getting-started/introduction) how to configure _Couper_.

This image contains a basic configuration to serve files from `/htdocs` directory.

```sh
$ docker run --rm -p 8080:8080 -v `pwd`:/htdocs avenga/couper
```

## Command

The entrypoint of the image is the `/couper` binary. The command is `run`.

Therefore `docker run avenga/couper` runs `/couper run -d /conf`.

The [directory argument](https://docs.couper.io/configuration/command-line#basic-options) allows you to mount multiple configuration files to the `/conf` directory.

You could also use other commands directly:

```sh
$ docker run avenga/couper version

$ docker run avenga/couper run -watch -p 8081
```

### Environment options

| Variable                             | Default | Description |
|:-------------------------------------| :------ | :---------- |
| COUPER_FILE                          | `couper.hcl` | Path to the configuration file. |
| COUPER_FILE_DIRECTORY                | `""`    | Path to the configuration files directory. |
| COUPER_ENVIRONMENT                   | `""`    | Name of environment in which Couper is currently running. |
| COUPER_ACCEPT_FORWARDED_URL          | `""`    | Which `X-Forwarded-*` request headers should be accepted to change the [request variables](https://docs.couper.io/configuration/variables#request) `url`, `origin`, `protocol`, `host`, `port`. Comma-separated list of values. Valid values: `proto`, `host`, `port`. |
| COUPER_DEFAULT_PORT                  | `8080`  | Sets the default port to the given value and does not override explicit `[host:port]` configurations from file. |
| COUPER_HEALTH_PATH                   | `/healthz` | Path for health-check requests for all servers and ports. |
| COUPER_HTTPS_DEV_PROXY               | `""`    | List of TLS port mappings to define the TLS listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname. |
| COUPER_NO_PROXY_FROM_ENV             | `false` | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). |
| COUPER_SECURE_COOKIES                | `""`    | If set to `"strip"`, the `Secure` flag is removed from all `Set-Cookie` HTTP header fields. |
| COUPER_WATCH                         | `false` | Set to `true` to watch for configuration file changes. |
| COUPER_WATCH_RETRIES                 | `5`     | Maximal retry count for configuration reloads which could not bind the configured port. |
| COUPER_WATCH_RETRY_DELAY             | `500ms` | Delay duration before next attempt if an error occurs. |
| COUPER_XFH                           | `false` | Global configurations which uses the `X-Forwarded-Host` header instead of the request host. |
| COUPER_CA_FILE                       | `""` | Option for adding the given PEM encoded ca-certificate to the existing system certificate pool for all outgoing connections. |
|                                      | | |
| COUPER_BETA_METRICS                  | `false`  | Option to enable the prometheus [metrics](https://github.com/avenga/couper/blob/master/docs/METRICS.md) exporter. |
| COUPER_BETA_METRICS_PORT             | `9090`   | Prometheus exporter listen port. |
| COUPER_BETA_SERVICE_NAME             | `couper` | The service name which applies to the `service_name` metric labels. |
|                                      | | |
| COUPER_REQUEST_ID_ACCEPT_FROM_HEADER | `""` | Name of a client request HTTP header field that transports the `request.id` which Couper takes for logging and transport to the backend (if configured). |
| COUPER_REQUEST_ID_BACKEND_HEADER     | `Couper-Request-ID` | Name of a HTTP header field which Couper uses to transport the `request.id` to the backend. |
| COUPER_REQUEST_ID_CLIENT_HEADER      | `Couper-Request-ID` | Name of a HTTP header field which Couper uses to transport the `request.id` to the client. |
| COUPER_REQUEST_ID_FORMAT             | `common` | If set to `uuid4` a rfc4122 uuid is used for `request.id` and related log fields. |
|                                      | | |
| COUPER_TIMING_IDLE_TIMEOUT           | `60s` | The maximum amount of time to wait for the next request on client connections when keep-alives are enabled. |
| COUPER_TIMING_READ_HEADER_TIMEOUT    | `10s` | The amount of time allowed to read client request headers. |
| COUPER_TIMING_SHUTDOWN_DELAY         | `0` | The amount of time the server is marked as unhealthy until calling server close finally. |
| COUPER_TIMING_SHUTDOWN_TIMEOUT       | `0` | The maximum amount of time allowed to close the server with all running connections. |
|                                      | | |
| COUPER_LOG_FORMAT                    | `common` | Can be set to `json` output which is the _container default_. |
| COUPER_LOG_LEVEL                     | `info` | Set the log-level to one of: `info`, `panic`, `fatal`, `error`, `warn`, `debug`, `trace`. |
| COUPER_LOG_PARENT_FIELD              | `""` | An option for `json` log format to add all log fields as child properties. |
| COUPER_LOG_PRETTY                    | `false` | Global option for `json` log format which pretty prints with basic key coloring. |
| COUPER_LOG_TYPE_VALUE                | `couper_daemon` | Value for the runtime log field `type`. |
| COUPER_SERVER_TIMING_HEADER          | `false` | If enabled, Couper includes an additional [Server-Timing](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Server-Timing) HTTP response header field detailing connection and transport relevant metrics for each backend request. |
| COUPER_ACCESS_LOG_REQUEST_HEADERS    | `User-Agent, Accept, Referer` | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_RESPONSE_HEADERS   | `Cache-Control, Content-Encoding, Content-Type, Location` | A comma separated list of header names whose values should be logged. |
| COUPER_ACCESS_LOG_TYPE_VALUE         | `couper_access` | Value for the log field `type`. |
| COUPER_BACKEND_LOG_REQUEST_HEADERS   | `User-Agent, Accept, Referer` | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_RESPONSE_HEADERS  | `Cache-Control, Content-Encoding, Content-Type, Location` | A comma separated list of header names whose values should be logged. |
| COUPER_BACKEND_LOG_TYPE_VALUE        | `couper_backend` | Value for the log field `type`. |
|                                      | | |
| COUPER_PPROF                         | `false` | Enables profiling. |
| COUPER_PPROF_PORT                    | `6060` | Port for profiling interface |
