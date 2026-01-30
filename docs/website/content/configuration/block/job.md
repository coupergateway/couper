---
title: 'Job (Beta)'
slug: 'job'
---

# Job (Beta)

The `beta_job` block lets you define recurring requests or sequences with an given interval. The job runs at startup
and for every interval and has an own log type: `job` which represents the starting point with an uid for tracing
purposes.

| Block name | Context                                               | Label    |
|:-----------|:------------------------------------------------------|:---------|
| `beta_job` | [Definitions Block](/configuration/block/definitions) | required |

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


{{< attributes >}}
[
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "Execution interval.",
    "name": "interval",
    "type": "duration"
  },
  {
    "default": "\"0s\"",
    "description": "Delays the first execution of the job after startup.",
    "name": "startup_delay",
    "type": "duration"
  }
]
{{< /attributes >}}

{{< duration >}}

{{< blocks >}}
[
  {
    "description": "Configures a [request](/configuration/block/request) (zero or more).",
    "name": "request"
  }
]
{{< /blocks >}}
