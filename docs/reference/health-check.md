# Configuration Reference ~ Health-Check

The Health-Check will answer with a HTTP status `200 OK` on every port with the
configured `health_path`.

The `health_path` can be configured via [Environment](environment.md),
[Command Line Interface](cli.md) or [Settings Block](blocks/settings.md). The path
defaults to `/healthz`.

As soon as the gateway instance will receive a `SIGINT` or `SIGTERM` the check will
return a HTTP status `500 Internal Server Error`.

For example, a [Shutdown Delay](environment.md) of `5s` allows the server to finish
all running requests and gives a load-balancer time to pick another gateway instance.
After this delay the server goes into shutdown mode with a [Shutdown Timeout](environment.md)
(deadline) of `5s` and no new requests will be accepted.

The shutdown timings default to `0` which means no delaying with development setups.
Both [Durations](config-types.md#duration) can be configured via [Environment](environment.md)
variable.

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Functions](functions.md)
* &#8674; [Modifiers](modifiers.md)
