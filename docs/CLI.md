- [Command Line Interface](#command-line-interface)
  - [Global Options](#global-options)

# Command Line Interface

Couper is build as binary called `couper` with the following commands:

| Command   | Description                                                                                                                                   |
| :-------- | :-------------------------------------------------------------------------------------------------------------------------------------------- |
| `run`     | Start the server with given configuration file.                                                                                               |
|           | _Note_: `run` options can also be configured with [settings](REFERENCE.md#settings-block) or related [environment variables](./../DOCKER.md). |
| `help`    | Print the usage for the given command: `help run`                                                                                             |
| `version` | Print the current version and build information.                                                                                              |

## Global Options

| Argument             | Default      | Environment                | Description                                                                             |
| :------------------- | :----------- | :------------------------- | :-------------------------------------------------------------------------------------- |
| `-f`                 | `couper.hcl` | `COUPER_FILE`              | File path to your Couper configuration file.                                            |
| `-watch`             | `false`      | `COUPER_WATCH`             | Watch for configuration file changes and reload on modifications.                       |
| `-watch-retries`     | `5`          | `COUPER_WATCH_RETRIES`     | Maximal retry count for configuration reloads which could not bind the configured port. |
| `-watch-retry-delay` | `500ms`      | `COUPER_WATCH_RETRY_DELAY` | Delay duration before next attempt if an error occurs.                                  |
| `-log-format`        | `common`     | `COUPER_LOG_FORMAT`        | Can be set to `json` output format.                                                     |
| `-log-pretty`        | `false`      | `COUPER_LOG_PRETTY`        | Option for `json` log format which pretty prints with basic key coloring.               |

_Note_: `log-format` and `log-pretty` also maps to [settings](REFERENCE.md#settings-block).

## Run Options

| Argument                | Default      | Environment                   | Description  |
| :---------------------- | :----------- | :---------------------------- | :----------- |
| `-accept-forwarded-url` | empty string | `COUPER_ACCEPT_FORWARDED_URL` | Which `X-Forwarded-*` request headers should be accepted to change the [variables](#variables) `request.url`, `request.origin`, `request.proto`, `request.host`, `request.port`. Comma-separated list of values. Valid values: `proto`, `host`, `port` |
