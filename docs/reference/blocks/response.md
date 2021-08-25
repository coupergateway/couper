# Response Block

The `response` block creates and sends a client response.

| Block name | Label    | Related blocks |
| ---------- | :------: | -------------- |
| `response` | &#10005; | [Endpoint Block](endpoint.md), [Error Handler Block](error-handler.md) |

## Attributes

| Attribute                       | Type                          | Default  | Description |
| ------------------------------- | ----------------------------- | :------: | ----------- |
| [`body`](../attributes.md)      | string                        | `""`     | Creates implicit default `Content-Type: text/plain` HTTP header field. |
| [`headers`](../attributes.md)   | [map](../config-types.md#map) | `{}`     | Same as [`set_response_headers`](../modifiers/set-response-headers.md) modifier. |
| [`json_body`](../attributes.md) | various                       | &#10005; | Creates implicit default `Content-Type: application/json` HTTP header field. |
| [`status`](../attributes.md)    | integer                       | `200`    | Sets response HTTP status code. |

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [Request Block](request.md)
* &#8674; [SAML Block](saml.md)
