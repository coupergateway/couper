# Websockets Block

The `websockets` block activates support for websocket connections in Couper.

| Block name   | Label    | Related blocks          |
| ------------ | :------: | ----------------------- |
| `websockets` | &#10005; | [Proxy Block](proxy.md) |

## Attributes

| Attribute                     | Type                                    | Default  | Description |
| ----------------------------- | --------------------------------------- | :------: | ----------- |
| [`timeout`](../attributes.md) | [duration](../config-types.md#duration) | &#10005; | The total deadline duration a websocket connection has to exists. |
| Modifiers                     | &#10005;                                | &#10005; | The `websockets` block supports [Header Modifiers](../modifiers.md#header-modifiers). |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [SPA Block](spa.md)
* &#8674; [API Block](api.md)
