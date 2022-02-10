- [Command Line Interface](#command-line-interface)
  - [Global Options](#global-options)
  - [Run Options](#run-options)

# Command Line Interface

Couper is build as binary called `couper` with the following commands:

| Command   | Description                                                                                                                                   |
|:----------|:----------------------------------------------------------------------------------------------------------------------------------------------|
| `run`     | Start the server with given configuration file.                                                                                               |
|           | _Note_: `run` options can also be configured with [settings](REFERENCE.md#settings-block) or related [environment variables](./../DOCKER.md). |
| `help`    | Print the usage for the given command: `help run`                                                                                             |
| `verify`  | Verify the syntax of the given configuration file.                                                                                            |
| `version` | Print the current version and build information.                                                                                              |

## Global Options

| Argument             | Default      | Environment                | Description                                                                                                                  |
|:---------------------|:-------------|:---------------------------|:-----------------------------------------------------------------------------------------------------------------------------|
| `-f`                 | `couper.hcl` | `COUPER_FILE`              | File path to your Couper configuration file.                                                                                 |
| `-d`                 | -            | `COUPER_FILE_DIRECTORY`    | File path to your Couper configuration files directory.                                                                      |
| `-watch`             | `false`      | `COUPER_WATCH`             | Watch for configuration file changes and reload on modifications.                                                            |
| `-watch-retries`     | `5`          | `COUPER_WATCH_RETRIES`     | Maximum retry count for configuration reloads which could not bind the configured port.                                      |
| `-watch-retry-delay` | `500ms`      | `COUPER_WATCH_RETRY_DELAY` | Delay duration before next attempt if an error occurs.                                                                       |
| `-log-format`        | `common`     | `COUPER_LOG_FORMAT`        | Can be set to `json` output format.                                                                                          |
| `-log-level`         | `info`       | `COUPER_LOG_LEVEL`         | Set the log-level to one of: `info`, `panic`, `fatal`, `error`, `warn`, `debug`, `trace`.                                    |
| `-log-pretty`        | `false`      | `COUPER_LOG_PRETTY`        | Option for `json` log format which pretty prints with basic key coloring.                                                    |
| `-ca-file`           | `""`         | `COUPER_CA_FILE`           | Option for adding the given PEM encoded ca-certificate to the existing system certificate pool for all outgoing connections. |

_Note_: `log-format`, `log-level` and `log-pretty` also map to [settings](REFERENCE.md#settings-block).

_Note_: Couper can be started with both, `-f` and `-d` arguments. The path of `-f <file>`
is the working directory of Couper. If `-d <dir>` argument is given without the `-f <file>`,
the path of `-d <dir>` is the working directory of Couper. A `couper.hcl` file inside the
`-d <dir>` is priorized over other files inside the `-d <dir>`, but not over the
`-f <file>`. Other files in the `-d <dir>` are loaded in alphabetical order. Example:

```sh
|- couper.hcl    # defined via `-f`                       <|
|- couper.d/     # defined via `-d`                        |
|  |- couper.hcl # patches the couper.hcl              <|  |
|  |- a.hcl      # patches the couper.d/couper.hcl  <|  |
|  |- z.hcl      # patches the couper.d/a.hcl        |
```

## Run Options

| Argument                | Default      | Environment                   | Description                                                                                                                                                                                                                           |
|:------------------------|:-------------|:------------------------------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-accept-forwarded-url` | empty string | `COUPER_ACCEPT_FORWARDED_URL` | Which `X-Forwarded-*` request headers should be accepted to change the [request variables](./REFERENCE.md#request) `url`, `origin`, `protocol`, `host`, `port`. Comma-separated list of values. Valid values: `proto`, `host`, `port` |
| `-https-dev-proxy`      | empty string | `COUPER_HTTPS_DEV_PROXY`      | List of tls port mappings to define the tls listen port and the target one. A self-signed certificate will be generated on the fly based on given hostname.                                                                           |
| `-beta-metrics`         | -            | `COUPER_BETA_METRICS`         | Option to enable the prometheus [metrics](./METRICS.md) exporter.                                                                                                                                                                     |
| `-beta-metrics-port`    | `9090`       | `COUPER_BETA_METRICS_PORT`    | Prometheus exporter listen port.                                                                                                                                                                                                      |
| `-beta-service-name`    | `couper`     | `COUPER_BETA_SERVICE_NAME`    | The service name which applies to the `service_name` metric labels.                                                                                                                                                                   |
