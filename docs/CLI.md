# Command Line Interface

Couper is build as binary called `couper` with the following commands:

| Command   | Description                                       |
|:----------|:--------------------------------------------------|
| `run`     | Start the server with given configuration file.   |
|           | *Note*: `run` options can also be configured with [settings](#settings-block) or related [environment variables](./../DOCKER.md).
| `help`    | Print the usage for the given command: `help run` |
| `version` | Print the current version and build information.  |

## Global Options

| Argument  | Default      | Environment       | Description                                                       |
|:----------|:-------------|:------------------|:------------------------------------------------------------------|
| `-f`      | `couper.hcl` | `COUPER_FILE`     | File path to your Couper configuration file.                      |
| `-watch`  | `false`      | `COUPER_WATCH`    | Watch for configuration file changes and reload on modifications. |
| `-watch-retries`  | `5`  | `COUPER_WATCH_RETRIES` | Maximal retry count for configuration reloads which could not bind the configured port. |
| `-watch-retry-delay`  | `500ms`  | `COUPER_WATCH_RETRY_DELAY` | Delay duration before next attempt if an error occurs. |
| `-log-format`  | `common` | `COUPER_LOG_FORMAT` | Can be set to `json` output format. |
| `-log-pretty`  | `false` | `COUPER_LOG_PRETTY`  | Option for `json` log format which pretty prints with basic key coloring. |

*Note*: `log-format` and `log-pretty` also maps to [settings](#settings-block).


