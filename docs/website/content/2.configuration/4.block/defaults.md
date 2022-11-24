# Defaults

The `defaults` block lets you define default values.

| Block name | Context | Label | Nested block(s) |
|:-----------|:--------|:------|:----------------|
| `defaults` | -       | -     | -               |


::attributes
---
values: [
  {
    "default": "",
    "description": "One or more environment variable assignments. Keys must be either identifiers or simple string expressions.",
    "name": "environment_variables",
    "type": "object"
  }
]

---
::

Examples:

- [`environment_variables`](https://github.com/avenga/couper-examples/blob/master/env-var/README.md)
