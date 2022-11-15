# Rate Limit (Beta)

Rate limiting protects backend services. It implements quota management used to avoid cascading failures or to spare resources.

| Block name        | Context                         | Label | Nested block(s) |
| :---------------- | :------------------------------ | :---- | :-------------- |
| `beta_rate_limit` | named [`backend` block](/configuration/block/backend)| -     | -               |

::attributes
---
values: [
  {
    "default": "\"wait\"",
    "description": "If `mode` is set to `block` and the rate limit is exceeded, the client request is immediately answered with HTTP status code `429` (Too Many Requests) and no backend request is made. If `mode` is set to `wait` and the rate limit is exceeded, the request waits for the next free rate limiting period.",
    "name": "mode",
    "type": "string"
  },
  {
    "default": "",
    "description": "Defines the number of allowed backend requests in a period.",
    "name": "per_period",
    "type": "number"
  },
  {
    "default": "",
    "description": "Defines the rate limit period.",
    "name": "period",
    "type": "duration"
  },
  {
    "default": "\"sliding\"",
    "description": "Defines the window of the period. A `fixed` window permits `per_period` requests within `period` after the first request to the parent backend. After the `period` has expired, another `per_period` request is permitted. The sliding window ensures that only `per_period` requests are sent in any interval of length period.",
    "name": "period_window",
    "type": "string"
  }
]

---
::

::duration
