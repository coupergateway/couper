# Rate Limit

Rate limiting protects backend services. It implements quota management and used to avoid cascading failures or to spare resources.

| Block name   | Context                         | Label | Nested block(s) |
| :----------- | :------------------------------ | :---- | :-------------- |
| `rate_limit` | [Backend Block](#backend-block) | -     | -               |

| Attribute(s)    | Type                  | Default | Description | Characteristic(s) | Example |
| :-------------- | :-------------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `period`        | [duration](#duration) | - | Defines the rate limit period. | &#9888; required, must not be negative or `0` (zero) | `period = "1m"` |
| `per_period`    | integer               | - | Defines the number of allowed backend requests in a period. | &#9888; required, must not be negative or `0` (zero) | `per_period = 100` |
| `period_window` | string                | `"sliding"` | Defines the window of the period. A `fixed` window starts always at `:00`, e.g. a period of `"1h"` starts at `10:00:00 a.m.`, `11:00:00 p.m.` etc. A `sliding` period is a period from the time of the client request that lies behind, e.g. `10:23:44-11:23:45 a.m.` if the request time is `11:23:45 a.m.` and `period = "1h"` is defined. | Allowed values: `"fixed"` or `"sliding"` | `period_window = "sliding"` |

**Note:** Anonymous backends (inline backends without label) cannot define `rate_limit` block(s).

**Note:** The number of `per_period` after (re)start of Couper is scaled down depending on the set `period` and `period_window` values. For example, if Couper starts at `12:34:56 a.m.`, the `period` is set to `1m`, `per_period` is set to `60` (avg. `1` request per `second`) and the `period_window` is set to `"fixed"`, only `3` backend request are allowed for the current period (`12:34:00-12:34:59 a.m.`).

**Note:** The usage of `period_window = "fixed"` may result in unexpected behavior. For example, the `period` is set to `1m` and the `per_period` is set to `60`, if Couper has to make 60 requests at `:58` to `:59` seconds in the current period window and then 60 requests from `:00` to `:01` seconds in the next period window, Couper would handle all the 120 requests in 2 seconds.
