# SPA

The `spa` blocks configure the Web serving for SPA assets. Can be defined multiple times as long as the `base_path`+`paths` is unique.

| Block name | Context                       | Label    | Nested block(s)           |
|:-----------|:------------------------------|:---------|:--------------------------|
| `spa`      | [Server Block](#server-block) | Optional | [CORS Block](#cors-block) |

| Attribute(s)             | Type           | Default | Description                                                                | Characteristic(s)                   | Example                                  |
|:-------------------------|:---------------|:--------|:---------------------------------------------------------------------------|:------------------------------------|:-----------------------------------------|
| `base_path`              | string         | -       | Configures the path prefix for all requests.                               | -                                   | `base_path = "/assets"`                  |
| `bootstrap_file`         | string         | -       | Location of the bootstrap file.                                            | &#9888; required                    | `bootstrap_file = "./htdocs/index.html"` |
| `paths`                  | tuple (string) | -       | List of SPA paths that need the bootstrap file.                            | &#9888; required                    | `paths = ["/app/**"]`                    |
| `access_control`         | tuple (string) | -       | Sets predefined [Access Control](#access-control) for `spa` block context. | -                                   | `access_control = ["foo"]`               |
| `disable_access_control` | tuple (string) | -       | Disables access controls by name.                                          | -                                   | `disable_access_control = ["foo"]`       |
| `custom_log_fields`      | object         | -       | Defines log fields for [Custom Logging](LOGS.md#custom-logging).           | &#9888; Inherited by nested blocks. | -                                        |
