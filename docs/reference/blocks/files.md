# Files Block

The `files` block configures the file serving.

| Block name | Label    | Related blocks |
| ---------- | -------- | -------------- |
| `files`    | &#10005; | [Server Block](server.md) |

## Nested blocks

* [CORS Block](cors.md)

## Attributes

| Attribute                                    | Type                            | Default     | Description |
| -------------------------------------------- | ------------------------------- | ----------- | ----------- |
| [`access_control`](../attributes.md)         | [list](../config-types.md#list) | `[]`        | Enables predefined [Access Control](../access-control.md) for the current block context. |
| [`base_path`](../attributes.md)              | string                          | `""`        | Configures the path prefix for all requests. |
| [`disable_access_control`](../attributes.md) | [list](../config-types.md#list) | `[]`        | Disables predefined [Access Control](../access-control.md) for the current block context. |
| [`document_root`](../attributes.md)          | string                          | `""`        | &#9888; Required. Sets the path to the document root. |
| [`error_file`](../attributes.md)             | string                          | `""`        | Sets the path to a custom error file template. |
| Modifiers                                    | &#10005;                        | &#10005;    | The `files` block supports [Response Header Modifiers](../modifiers.md#response-header-modifiers). |

## Examples

* [File and Web Serving](../examples.md#file-and-web-serving)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Error Handler Block](error-handler.md)
* &#8674; [JWT Block](jwt.md)
