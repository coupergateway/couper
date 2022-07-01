### Server

The `server` block is one of the root configuration blocks of Couper's configuration file.

| Block name | Context | Label    | Nested block(s)                                                                                                                                       |
|:-----------|:--------|:---------|:------------------------------------------------------------------------------------------------------------------------------------------------------|
| `server`   | -       | optional | [CORS Block](#cors-block), [Files Block](#files-block), [SPA Block(s)](#spa-block) , [API Block(s)](#api-block), [Endpoint Block(s)](#endpoint-block) |

| Attribute(s)             | Type           | Default      | Description                                                                   | Characteristic(s)                                                                                                             | Example                                     |
|:-------------------------|:---------------|:-------------|:------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------|
| `base_path`              | string         | -            | Configures the path prefix for all requests.                                  | &#9888; Inherited by nested blocks.                                                                                           | `base_path = "/api"`                        |
| `hosts`                  | tuple (string) | `["*:8080"]` | -                                                                             | &#9888; required, if there is more than one `server` block. &#9888; Only one `hosts` attribute per `server` block is allowed. | `hosts = ["example.com", "localhost:9090"]` |
| `error_file`             | string         | -            | Location of the error file template.                                          | -                                                                                                                             | `error_file = "./my_error_page.html"`       |
| `access_control`         | tuple (string) | -            | Sets predefined [Access Control](#access-control) for `server` block context. | &#9888; Inherited by nested blocks.                                                                                           | `access_control = ["foo"]`                  |
| `disable_access_control` | tuple (string) | -            | Disables access controls by name.                                             | -                                                                                                                             | `disable_access_control = ["foo"]`          |
| `custom_log_fields`      | object         | -            | Defines log fields for [Custom Logging](LOGS.md#custom-logging).              | &#9888; Inherited by nested blocks.                                                                                           | -                                           |
