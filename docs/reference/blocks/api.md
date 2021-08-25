# API Block

The `api` block bundles [Endpoint Blocks](endpoint.md) under a certain `base_path`.

| Block name | Label               | Related blocks |
| ---------- | ------------------- | -------------- |
| `api`      | &#10003; (optional) | [Server Block](server.md) |

```diff
! If an error occurred in a nested "endpoint" block, the response gets processed as JSON error with an error body payload. This can be customized via the "error_file" attribute.
```

## Nested blocks

* [CORS Block](cors.md)
* [Endpoint Block(s)](endpoint.md)

## Attributes

| Attribute                                    | Type                            | Default     | Description |
| -------------------------------------------- | ------------------------------- | ----------- | ----------- |
| [`access_control`](../attributes.md)         | [list](../config-types.md#list) | `[]`        | Enables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`base_path`](../attributes.md)              | string                          | `""`        | Configures the path prefix for all requests. |
| [`disable_access_control`](../attributes.md) | [list](../config-types.md#list) | `[]`        | Disables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`error_file`](../attributes.md)             | string                          | `""`        | Sets the path to a custom error file template. |
| Modifiers                                    | &#10005;                        | &#10005;    | The `api` block supports [Response Header Modifiers](../modifiers.md#response-header-modifiers). |

```diff
! The combination of the api's "base_path" and the nested endpoint's "label" must be unique.
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Websockets Block](websockets.md)
* &#8674; [Backend Block](backend.md)
