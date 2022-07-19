# Rate Limit (Beta)

Rate limiting protects backend services. It implements quota management and used to avoid cascading failures or to spare resources.

| Block name        | Context                         | Label | Nested block(s) |
| :---------------- | :------------------------------ | :---- | :-------------- |
| `beta_rate_limit` | [Backend Block](#backend-block) | -     | -               |

| Attribute(s)    | Type                  | Default | Description | Characteristic(s) | Example |
| :-------------- | :-------------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `period`        | [duration](#duration) | - | Defines the rate limit period. | &#9888; required, must not be negative or `0` (zero) | `period = "1m"` |
| `per_period`    | integer               | - | Defines the number of allowed backend requests in a period. | &#9888; required, must not be negative or `0` (zero) | `per_period = 100` |
| `period_window` | string                | `"sliding"` | Defines the window of the period. A `fixed` window permits `per_period` requests within `period` after the first request to the parent backend. After the `period` has expired, another `per_period` requests are permitted. The `sliding` window ensures that only `per_period` requests are sent in any interval of length `period`. | Allowed values: `"fixed"` or `"sliding"` | `period_window = "sliding"` |
| `mode`          | string                | `"wait"` | If `mode` is set to `block` and the rate limit is exceeded, the client request is immediately answered with HTTP status code `429 Too Many Requests` and no backend request is made. If `mode` is set to `wait` and the rate limit is exceeded, the request waits for the next free rate limiting period. | Allowed values: `"wait"` or `"block"` | `mode = "wait"` |

**Note:** Anonymous backends (inline backends without label) cannot define `beta_rate_limit` block(s).
