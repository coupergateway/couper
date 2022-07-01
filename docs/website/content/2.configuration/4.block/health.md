# Health

Defines a recurring health check request for its backend. Results can be obtained via the [`backends.<label>.health` variables](#backends).
Changes in health states and related requests will be logged. Default User-Agent will be `Couper / <version> health-check` if not provided
via `headers` attribute. An unhealthy backend will return with a [`backend_unhealthy`](ERRORS.md#api-error-types) error.

| Block name    | Context                           | Label | Nested block |
|:--------------|:----------------------------------|:------|:-------------|
| `beta_health` | [`backend` block](#backend-block) | –     |              |

| Attributes          | Type                  | Default           | Description                                        | Characteristics       | Example                             |
|:--------------------|:----------------------|:------------------|:---------------------------------------------------|:----------------------|:------------------------------------|
| `expected_status`   | tuple (number)        | `[200, 204, 301]` | wanted response status code                        |                       | `expected_status = [418]`           |
| `expected_text`     | string                | –                 | text response body must contain                    |                       | `expected_text = "alive"`           |
| `failure_threshold` | number                | `2`               | failed checks needed to consider backend unhealthy |                       | `failure_threshold = 3`             |
| `headers`           | object                | –                 | request headers                                    |                       | `headers = {User-Agent = "health"}` |
| `interval`          | [duration](#duration) | `"1s"`            | time interval for recheck                          |                       | `timeout = "5s"`                    |
| `path`              | string                | –                 | URL path/query on backend host                     |                       | `path = "/health"`                  |
| `timeout`           | [duration](#duration) | `"2s"`            | maximum allowed time limit                         | bounded by `interval` | `timeout = "3s"`                    |
