# Endpoint Block

An `endpoint` block defines an entry point of Couper. Depending on where an endpoint
is placed in the configuration file, the entry point is defined by `base_path` attributes
of the parent blocks (e.g. [Server Block](server.md) or [API Block](api.md)) and
the `label` of the current `endpoint`.

```diff
! The "path" attribute changes the path for the outgoing request.
```

```diff
! Each "endpoint" block must produce an explicit or implicit client response via a "response" or "proxy" block.
```

**See also:**

* [Proxy Block](proxy.md)
* [Response Block](response.md)

| Block name | Label               | Related blocks |
| ---------- | ------------------- | -------------- |
| `endpoint` | &#10003; (required) | [API Block](api.md), [Server Block](server.md) |

## Nested blocks

* [Proxy Block(s)](proxy.md)
* [Request Block(s)](request.md)
* [Response Block](response.md)

## Attributes

| Attribute                                    | Type                            | Default  | Description |
| -------------------------------------------- | ------------------------------- | -------- | ----------- |
| [`access_control`](../attributes.md)         | [list](../config-types.md#list) | `[]`     | Enables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`disable_access_control`](../attributes.md) | [list](../config-types.md#list) | `[]`     | Disables predefined [Access Control](../access-control.md) for the current block context. Inherited by nested blocks. |
| [`error_file`](../attributes.md)             | string                          | `""`     | Sets the path to a custom error file template. |
| [`path`](../attributes.md)                   | string                          | `""`     | Changeable part of the upstream URL. Changes the path suffix of the outgoing request URL. |
| [`request_body_limit`](../attributes.md)     | [size](../config-types.md#size) | `64MiB`  | Configures the maximum buffer size while accessing [`request.form_body`](../variables/request.md) or [`request.json_body`](../variables/request.md) content. |
| Modifiers                                    | &#10005;                        | &#10005; | The `endpoint` block supports [Modifiers](../modifiers.md). |
| Parameters                                   | &#10005;                        | &#10005; | The `endpoint` block supports [Parameters](../parameters.md). |

## Examples

* [Exposing APIs](../examples.md#exposing-apis)
* [Path Routing and Mapping](../examples.md#path-routing-and-mapping)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Definitions Block](definitions.md)
* &#8674; [Error Handler Block](error-handler.md)
