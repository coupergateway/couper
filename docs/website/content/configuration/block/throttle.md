---
title: 'Throttle'
slug: 'throttle'
---

# Throttle

Throttling protects backend services by limiting the number of requests forwarded to an origin within a given time period. This helps avoid cascading failures or spare resources on upstream services with known capacity limits.

If the throttle limit is exceeded, Couper either queues the request until the next period (`mode = "wait"`, default) or immediately responds with HTTP status `429` (`mode = "block"`). Exceeded limits in `block` mode can be handled with an [`error_handler`](/configuration/block/error_handler) using the [`backend_throttle_exceeded`](/configuration/error-handling#endpoint-error-types) error type.

| Block name | Context                                               | Label    |
|:-----------|:------------------------------------------------------|:---------|
| `throttle` | named [`backend` block](/configuration/block/backend) | no label |

## Example: Sliding window (wait mode)

Allows 100 requests per minute to the backend. Excess requests wait for a free slot:

```hcl
definitions {
  backend "my_api" {
    origin = "https://api.example.com"

    throttle {
      period     = "1m"
      per_period = 100
    }
  }
}
```

## Example: Fixed window (block mode)

Allows 10 requests per second. Excess requests are immediately rejected with HTTP `429`:

```hcl
definitions {
  backend "strict_api" {
    origin = "https://api.example.com"

    throttle {
      period        = "1s"
      per_period    = 10
      period_window = "fixed"
      mode          = "block"
    }
  }
}

server {
  endpoint "/strict" {
    proxy {
      backend = "strict_api"
    }

    error_handler "backend_throttle_exceeded" {
      response {
        status = 429
        headers = {
          retry-after = "1"
        }
        json_body = {
          error = "rate limit exceeded, try again later"
        }
      }
    }
  }
}
```

{{< attributes >}}
[
  {
    "default": "\"wait\"",
    "description": "If `mode` is set to `block` and the throttle limit is exceeded, the client request is immediately answered with HTTP status code `429` (Too Many Requests) and no backend request is made. If `mode` is set to `wait` and the throttle limit is exceeded, the request waits for the next free throttling period.",
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
    "description": "Defines the throttle period.",
    "name": "period",
    "type": "duration"
  },
  {
    "default": "\"sliding\"",
    "description": "Defines the window of the period. A `fixed` window permits `per_period` requests within `period` after the first request to the parent backend. After the `period` has expired, another `per_period` request is permitted. The sliding window ensures that only `per_period` requests are sent in any interval of length `period`.",
    "name": "period_window",
    "type": "string"
  }
]
{{< /attributes >}}

{{< duration >}}
