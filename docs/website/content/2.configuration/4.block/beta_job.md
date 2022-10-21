# Beta Job

Use the `beta_job` block to execute a job periodically.

| Block name | Context                          | Label            | Nested block                |
|:-----------|:---------------------------------|:-----------------|:----------------------------|
| `beta_job` | [Definitions Block](definitions) | &#9888; required | [Request Block(s)](request) |

::attributes
---
values: [
  {
    "default": "",
    "description": "time interval for execution of a job",
    "name": "interval",
    "type": "string"
  },
  {
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging)",
    "name": "custom_log_fields",
    "type": "object"
  }
]

---
::

## Example

```hcl
...

definitions {
  beta_job "update_data" {
    # Execute once at the start of Couper and than minutely
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

    openapi {
      file = "write-api.openapi.yaml"
    }
  }
}
```
