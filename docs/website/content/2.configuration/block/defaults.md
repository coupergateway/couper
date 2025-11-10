# Defaults

The `defaults` block lets you define default values.

| Block name | Context | Label    |
|:-----------|:--------|:---------|
| `defaults` | -       | no label |


{{< attributes >}}
[
  {
    "default": "",
    "description": "One or more environment variable assignments. Keys must be either identifiers or simple string expressions.",
    "name": "environment_variables",
    "type": "object"
  }
]
{{< /attributes >}}

Examples:

- [`environment_variables`](https://github.com/coupergateway/couper-examples/blob/master/env-var/README.md)
