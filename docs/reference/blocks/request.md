# Request Block

The `request` block creates and executes a request to a backend service.

| Block name | Label               | Related blocks |
| ---------- | ------------------- | -------------- |
| `request`  | &#10003; (optional) | [Endpoint Block](endpoint.md), [Error Handler Block](error-handler.md) |

```diff
! A "proxy" or "request" block w/o a label has an implicit label "default". Only one block with label "default" per endpoint block is allowed.
```

```diff
! Multiple "proxy" and "request" blocks are executed in parallel.
```

## Nested blocks

* [Backend Block](backend.md)

## Attributes

| Attribute                          | Type                          | Default  | Description |
| ---------------------------------- | ----------------------------- | -------- | ----------- |
| [`body`](../attributes.md)         | string                        | `""`     | Creates implicit default `Content-Type: text/plain` HTTP header field. |
| [`backend`](../attributes.md)      | string                        | `""`     | A [Backend Block](backend.md) reference, defined in [Definitions Block](definitions.md). |
| [`form_body`](../attributes.md)    | [map](../config-types.md#map) | `{}`     | Creates implicit default `Content-Type: application/x-www-form-urlencoded` HTTP header field. |
| [`headers`](../attributes.md)      | [map](../config-types.md#map) | `{}`     | Same as [`set_request_headers`](../modifiers/set-request-headers.md) modifier. |
| [`json_body`](../attributes.md)    | various                       | &#10005; | Creates implicit default `Content-Type: application/json` HTTP header field. |
| [`method`](../attributes.md)       | string                        | `"GET"`  | Defines the request HTTP method. |
| [`query_params`](../attributes.md) | [map](../config-types.md#map) | `{}`     | Same as [`set_query_params`](../parameters/set-query-params.md) parameter. |
| [`url`](../attributes.md)          | string                        | `""`     | Shortcut for an [`origin`](../attributes.md) attribute inside a [Backend Block](backend.md) and the `path` attribute of current `request` block. |

```diff
! To be able to execute a request the "request" block needs a "backend" block, "backend" block reference or an "url" attribute.
```

```diff
! If defined both, the "url" attribute and an "origin" attribute in a "backend" block, the host part of the URL must be the same as in the "origin".
```

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Proxy Block](proxy.md)
* &#8674; [Response Block](response.md)
