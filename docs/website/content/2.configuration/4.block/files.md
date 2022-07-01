# Files

The `files` blocks configure the file serving. Can be defined multiple times as long as the `base_path` is unique.

| Block name | Context                       | Label    | Nested block(s)           |
|:-----------|:------------------------------|:---------|:--------------------------|
| `files`    | [Server Block](#server-block) | Optional | [CORS Block](#cors-block) |

| Attribute(s)             | Type           | Default | Description                                                                  | Characteristic(s)                   | Example                            |
|:-------------------------|:---------------|:--------|:-----------------------------------------------------------------------------|:------------------------------------|:-----------------------------------|
| `base_path`              | string         | -       | Configures the path prefix for all requests.                                 | -                                   | `base_path = "/files"`             |
| `document_root`          | string         | -       | Location of the document root.                                               | &#9888; required                    | `document_root = "./htdocs"`       |
| `error_file`             | string         | -       | Location of the error file template.                                         | -                                   | -                                  |
| `access_control`         | tuple (string) | -       | Sets predefined [Access Control](#access-control) for `files` block context. | -                                   | `access_control = ["foo"]`         |
| `disable_access_control` | tuple (string) | -       | Disables access controls by name.                                            | -                                   | `disable_access_control = ["foo"]` |
| `custom_log_fields`      | object         | -       | Defines log fields for [Custom Logging](LOGS.md#custom-logging).             | &#9888; Inherited by nested blocks. | -                                  |
