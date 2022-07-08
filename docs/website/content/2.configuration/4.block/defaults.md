# Defaults

The `defaults` block lets you define default values.

| Block name | Context | Label | Nested block(s) |
|:-----------|:--------|:------|:----------------|
| `defaults` | -       | -     | -               |


::attributes
---
values: [
  {
    "name": "environment_variables",
    "type": "object",
    "default": "",
    "description": "One or more environment variable assignments"
  }
]

---
::

Examples:

- [`environment_variables`](https://github.com/avenga/couper-examples/blob/master/env-var/README.md)
