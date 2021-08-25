# Server Block

The `server` block is one of the root configuration blocks of Couper's configuration
file.

| Block name | Label               | Related blocks |
| ---------- | ------------------- | :------------: |
| `server`   | &#10003; (required) | &#10005;       |

## Nested blocks

* [API Block(s)](api.md)
* [CORS Block](cors.md)
* [Endpoint Block(s)](endpoint.md)
* [Files Block](files.md)
* [SPA Block](spa.md)

## Attributes

| Attribute                                    | Type                            | Default     | Description |
| -------------------------------------------- | ------------------------------- | ----------- | ----------- |
| [`access_control`](../attributes.md)         | [list](../config-types.md#list) | `[]`        | Enables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`base_path`](../attributes.md)              | string                          | `""`        | Configures the path prefix for all requests. Inherited by nested blocks. |
| [`disable_access_control`](../attributes.md) | [list](../config-types.md#list) | `[]`        | Disables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`error_file`](../attributes.md)             | string                          | `""`        | Sets the path to a custom error file template. |
| [`hosts`](../attributes.md)                  | [list](../config-types.md#list) | `[":8080"]` | Defines the hosts and ports where Couper should listen to. Definition of `hosts` is required, if there is more than one "server" blocks defined. |
| Modifiers                                    | &#10005;                        | &#10005;    | The `server` block supports [Response Header Modifiers](../modifiers.md#response-header-modifiers). |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [SAML Block](saml.md)
* &#8674; [Settings Block](settings.md)
