# Server

The `server` block is one of the root configuration blocks of Couper's configuration file.

| Block name | Context | Label    | Nested block(s)                                                                                                                                       |
|:-----------|:--------|:---------|:------------------------------------------------------------------------------------------------------------------------------------------------------|
| `server`   | -       | optional | [CORS Block](cors), [Files Block](files), [SPA Block(s)](spa) , [API Block(s)](api), [Endpoint Block(s)](endpoint) |

| Attribute(s)             | Type           | Default      | Description                                                                   | Characteristic(s)                                                                                                             | Example                                     |
|:-------------------------|:---------------|:-------------|:------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------|
| `base_path`              | string         | -            | Configures the path prefix for all requests.                                  | &#9888; Inherited by nested blocks.                                                                                           | `base_path = "/api"`                        |
| `hosts`                  | tuple (string) | `["*:8080"]` | -                                                                             | &#9888; required, if there is more than one `server` block. &#9888; Only one `hosts` attribute per `server` block is allowed. | `hosts = ["example.com", "localhost:9090"]` |
| `error_file`             | string         | -            | Location of the error file template.                                          | -                                                                                                                             | `error_file = "./my_error_page.html"`       |
| `access_control`         | tuple (string) | -            | Sets predefined [access control](../access-control) for `server` block context. | &#9888; Inherited by nested blocks.                                                                                           | `access_control = ["foo"]`                  |
| `disable_access_control` | tuple (string) | -            | Disables access controls by name.                                             | -                                                                                                                             | `disable_access_control = ["foo"]`          |
| `custom_log_fields`      | object         | -            | Defines log fields for [Custom Logging](/observation/logging#custom-logging).              | &#9888; Inherited by nested blocks.                                                                                           | -                                           |
