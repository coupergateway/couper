---
title: 'Rate Limiter (Beta)'
slug: 'rate_limiter'
---

# Rate Limiter (Beta)

| Block name          | Context                                               | Label    |
|:--------------------|:------------------------------------------------------|:---------|
| `beta_rate_limiter` | [Definitions Block](/configuration/block/definitions) | required |

Rate Limier is a beta feature that lets you configure rate limiting for your gateway. It is defined in the
[`definitions` block](/configuration/block/definitions) and can be referenced in all access control attributes
by its required _label_.

## Example

```hcl
server {
  endpoint "/ip_rate/**" {
    access_control = ["ip_rate"]
    response {
      json_body = {
        ok = true
      }
    }
  }
  
  endpoint "/jwt_rate/**" {
    access_control = ["my_jwt", "jwt_rate"]
    response {
      json_body = {
        ok = true
      }
    }
  }
}

definitions {
  beta_rate_limiter "ip_rate" {
    period = "60s"
    per_period = 5
    period_window = "sliding"
    key = request.remote_ip
  }
  
  
  jwt "my_jwt" { 
    #...
  }
  
  beta_rate_limiter "jwt_rate" {
    period = "100s"
    per_period = 1
    # period_window = "fixed"
    key = request.context.my_jwt.sub
  }
}
```

{{< attributes >}}
---
values: [
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "The expression defining which key to be used to identify a visitor.",
    "name": "key",
    "type": "string"
  },
  {
    "default": "",
    "description": "Defines the number of allowed requests in a period.",
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
    "description": "Defines the window of the period. A `fixed` window permits `per_period` requests within `period`. After the `period` has expired, another `per_period` request is permitted. The sliding window ensures that only `per_period` requests are sent in any interval of length `period`.",
    "name": "period_window",
    "type": "string"
  }
]

---
{{< /attributes >}}

{{< duration >}}

{{< blocks >}}
[
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]
{{< /blocks >}}
