# Couper 2

Couper 2 is designed to support developers building and operating API-driven Web projects by offering security and observability functionality in a  lightweight API gateway component.

_For additional information, tutorials and documentation please visit the [couper repository](https://github.com/avenga/couper)._

## Usage

Couper requires a configuration file which have to be provided on start.
See our [documentation](https://github.com/avenga/couper/trees/master/docs/) how to configure _couper_.

`docker run --rm -p 8080:8080 -v "$(pwd)":/conf avenga/couper`

### Environment options

| Variable  | Default  | Description  |
|---        |---       |---           |
| COUPER_PORT   | `8080`    | Sets the default port to the given value and does not override explicit `[host:port]` configurations from file. |
| COUPER_XFH    | `false`   | Global configurations which uses the `Forwarded-Host` header instead of the request host.   |
| COUPER_LOG_FORMAT | `common`  | Can be set to `json` output which is the _container default_. |
