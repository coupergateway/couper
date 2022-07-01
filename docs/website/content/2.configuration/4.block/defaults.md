# Defaults

The `defaults` block lets you define default values.

| Block name | Context | Label | Nested block(s) |
|:-----------|:--------|:------|:----------------|
| `defaults` | -       | -     | -               |

| Attribute(s)            | Type            | Default | Description                                 | Characteristic(s) | Example                                                        |
|:------------------------|:----------------|:--------|:--------------------------------------------|:------------------|:---------------------------------------------------------------|
| `environment_variables` | object (string) | –       | One or more environment variable assigments | -                 | `environment_variables = {ORIGIN = "https://httpbin.org" ...}` |

Examples:

- [`environment_variables`](https://github.com/avenga/couper-examples/blob/master/env-var/README.md).
