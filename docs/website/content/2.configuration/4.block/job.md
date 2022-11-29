# Job (Beta)

The `beta_job` block lets you define recurring requests or sequences with an given interval. The job runs at startup
and for every interval and has an own log type: `job` which represents the starting point with an uid for tracing
purposes.

| Block name | Context                          | Label            | Nested block                |
|:-----------|:---------------------------------|:-----------------|:----------------------------|
| `beta_job` | [Definitions Block](definitions) | required         | [Request Block(s)](request) |

## Example

```hcl
# ...
definitions {
  beta_job "update_data" {
    # Execute once at the start of Couper and then every minute
    interval = "1m"

    request "origin" {
      url             = "/api/v1/exports/data"
      backend         = "read"
      expected_status = [200]
    }

    request "update" {
      url     = "/update"
      body    = backend_responses.origin.body
      backend = "write"
    }
  }

  backend "read" {
    origin     = "${env.MY_ORIGIN}"
    basic_auth = env.MY_AUTH
  }

  backend "write" {
    origin = "${env.ORIGIN_DATABASE}"
  }
}
```


::attributes
---
values: [
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "\"60s\"",
    "description": "Execution interval",
    "name": "interval",
    "type": "duration"
  },
  {
    "default": "",
    "description": "[`request`](request) block definition.",
    "name": "request",
    "type": "object"
  }
]

---
::

::duration
---
---
::
