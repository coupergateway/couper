# Rate Limit (Beta)

Rate limiting protects backend services. It implements quota management and used to avoid cascading failures or to spare resources.

| Block name        | Context                         | Label | Nested block(s) |
| :---------------- | :------------------------------ | :---- | :-------------- |
| `beta_rate_limit` | [Backend Block](#backend-block) | -     | -               |

| Attribute(s)    | Type                  | Default | Description | Characteristic(s) | Example |
| :-------------- | :-------------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `period`        | [duration](#duration) | - | Defines the rate limit period. | &#9888; required, must not be negative or `0` (zero) | `period = "1m"` |
| `per_period`    | integer               | - | Defines the number of allowed backend requests in a period. | &#9888; required, must not be negative or `0` (zero) | `per_period = 100` |
| `period_window` | string                | `"sliding"` | Defines the window of the period. A `fixed` window always starts with the first request to the parent `backend`. A `sliding` period is a period from the time of the client request that lies behind, e.g. `10:23:44-11:23:45 a.m.` if the request time is `11:23:45 a.m.` and `period = "1h"` is defined. | Allowed values: `"fixed"` or `"sliding"` | `period_window = "sliding"` |
| `mode`          | string                | `"wait"` | If `mode` is set to `block` and the rate limit is exceeded, the client request is immediately answered with HTTP status code `429 Too Many Requests` and no backend request is made. If `mode` is set to `wait` and the rate limit is exceeded, the request waits for the next free rate limiting period. | Allowed values: `"wait"` or `"block"` | `mode = "wait"` |

**Note:** Anonymous backends (inline backends without label) cannot define `beta_rate_limit` block(s).

**Note:** The number of `per_period` after (re)start of Couper is scaled down depending on the set `period` and `period_window` values. For example, if Couper starts at `12:34:56 a.m.`, the `period` is set to `1m`, `per_period` is set to `60` (avg. `1` request per `second`) and the `period_window` is set to `"sliding"`, only `3` backend request are allowed for the current period (`12:34:00-12:34:59 a.m.`).
