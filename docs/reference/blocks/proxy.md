# Proxy Block

The `proxy` block creates and executes a proxy request to a backend service.

| Block name | Label               | Related blocks |
| ---------- | ------------------- | -------------- |
| `proxy`    | &#10003; (optional) | [Endpoint Block](endpoint.md), [Error Handler Block](error-handler.md) |

```diff
! A "proxy" or "request" block w/o a label has an implicit label "default". Only one block with label "default" per endpoint block is allowed.
```

```diff
! Multiple "proxy" and "request" blocks are executed in parallel.
```

**See also:**

* [Endpoint Block](endpoint.md)
* [Request Block](request.md)

## Nested blocks

* [Backend Block](backend.md)
* [Websockets Block](websockets.md)

## Attributes

| Attribute                        | Type     | Default  | Description |
| -------------------------------- | -------- | -------- | ----------- |
| [`backend`](../attributes.md)    | string   | `""`     | A [Backend Block](backend.md) reference, defined in [Definitions Block](definitions.md). |
| [`path`](../attributes.md)       | string   | `""`     | Changeable part of the upstream URL. Changes the path suffix of the outgoing request URL. |
| [`url`](../attributes.md)        | string   | `""`     | Shortcut for an [`origin`](../attributes.md) attribute inside a [Backend Block](backend.md) and the `path` attribute of current `proxy` block. |
| [`websockets`](../attributes.md) | bool     | `false`  | Enables websockets support. |
| Modifiers                        | &#10005; | &#10005; | The `endpoint` block supports [Header Modifiers](../modifiers.md#header-modifiers). |
| Parameters                       | &#10005; | &#10005; | The `endpoint` block supports [Parameters](../parameters.md). |

```diff
! To be able to execute a request the "proxy" block needs a "backend" block, "backend" block reference or an "url" attribute.
```

```diff
! If defined both, the "url" attribute and an "origin" attribute in a "backend" block, the host part of the URL must be the same as in the "origin".
```

```diff
! Either, "websockets" attribute or block is allowed.
```

```diff
! The "websockets" are only allowed in the "default" "proxy" block. Other "proxy", "request" or "response" blocks are not allowed in the current "endpoint" block.
```

**See also:**

* [Endpoint Block](endpoint.md)
* [Request Block](request.md)
* [Response Block](response.md)
* [Websockets Block](websockets.md)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [OpenAPI Block](openapi.md)
* &#8674; [Request Block](request.md)
