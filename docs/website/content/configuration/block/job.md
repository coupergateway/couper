---
title: 'Job'
slug: 'job'
---

# Job

The `job` block lets you define recurring requests or sequences with a given interval. The job runs at startup
and then at every interval and has its own log type: `couper_job`, which represents the starting point with a UID for tracing
purposes. Older configurations may still use the legacy block name `beta_job`, which remains supported as an alias.

| Block name | Context                                               | Label    |
| :--------- | :---------------------------------------------------- | :------- |
| `job`      | [Definitions Block](/configuration/block/definitions) | required |

## Example

```hcl
# ...
definitions {
  job "update_data" {
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
