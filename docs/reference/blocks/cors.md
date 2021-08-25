# Cross-Origin Resource Sharing (CORS) Block

The `cors` block configures the CORS behavior in Couper.

| Block name | Label    | Related blocks |
| ---------- | :------: | -------------- |
| `cors`     | &#10005; | [API Block](api.md), [Files Block](files.md), [Server Block](server.md), [SPA Block](spa.md) |

```diff
! Overrides the CORS behavior of the parent block.
```

## Attributes

| Attribute                               | Type                                      | Default  | Description |
| --------------------------------------- | ----------------------------------------- | :------: | ----------- |
| [`allow_credentials`](../attributes.md) | bool                                      | `false`  | Set to `true` if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` HTTP header fields). |
| [`allowed_origins`](../attributes.md)   | [list](../config-types.md#list) or string | &#10005; | The `"*"` allowes all origins, a list configures specific origins. |
| [`disable`](../attributes.md)           | bool                                      | `false`  | Set to `true` to disable the inheritance of CORS from the parent block. |
| [`max_age`](../attributes.md)           | [duration](../config-types.md#duration)   | &#10005; | Indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response HTTP header fields. &#9888; Can be cached. |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Basic Auth Block](basic-auth.md)
* &#8674; [Defaults Block](defaults.md)
