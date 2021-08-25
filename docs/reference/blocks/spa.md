# SPA Block

The `spa` block configures the Web serving for SPA assets.

| Block name | Label    | Related blocks |
| ---------- | -------- | -------------- |
| `spa`      | &#10005; | [Server Block](server.md) |

## Nested blocks

* [CORS Block](cors.md)

## Attributes

| Attribute                                    | Type                            | Default     | Description |
| -------------------------------------------- | ------------------------------- | ----------- | ----------- |
| [`access_control`](../attributes.md)         | [list](../config-types.md#list) | `[]`        | Enables predefined [Access Control](../access-control.md) for the current block context. |
| [`base_path`](../attributes.md)              | string                          | `""`        | Configures the path prefix for all requests. |
| [`bootstrap_file`](../attributes.md)         | string                          | `""`        | &#9888; Required. Configures the path to the bootstrap file. |
| [`disable_access_control`](../attributes.md) | [list](../config-types.md#list) | `[]`        | Disables predefined [Access Control](../access-control.md) for the current block context. |
| [`paths`](../attributes.md)                  | [list](../config-types.md#list) | `[]`        | &#9888; Required. Sets the list of SPA paths that need the bootstrap file. |
| Modifiers                                    | &#10005;                        | &#10005;    | The `spa` block supports [Response Header Modifiers](../modifiers.md#response-header-modifiers). |

## Examples

* [File and Web Serving](../examples.md#file-and-web-serving)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Settings Block](settings.md)
* &#8674; [Websockets Block](websockets.md)
