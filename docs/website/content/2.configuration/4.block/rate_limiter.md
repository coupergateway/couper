
::attributes
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
::

::blocks
---
values: [
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]

---
::
