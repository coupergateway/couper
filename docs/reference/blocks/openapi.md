# OpenAPI Block

The `openapi` block configures the backends proxy behavior to validate outgoing
and incoming requests to and from the origin. Prevents origins from invalid requests
and Couper from invalid answers. The validation based on the
[OpenAPI 3 standard](https://www.openapis.org).

| Block name | Label    | Related blocks              |
| ---------- | :------: | --------------------------- |
| `openapi`  | &#10005; | [Backend Block](backend.md) |

## Attributes

| Attribute                                        | Type   | Default | Description |
| ------------------------------------------------ | ------ | ------- | ----------- |
| [`file`](../attributes.md)                       | string | `""`    | &#9888; Required. OpenAPI [YAML](https://en.wikipedia.org/wiki/YAML) definition file. |
| [`ignore_request_violations`](../attributes.md)  | bool   | `false` | Log request validation results, skip error handling. |
| [`ignore_response_violations`](../attributes.md) | bool   | `false` | Log response validation results, skip error handling. |

```diff
! While ignoring request violations an invalid "method" or "path" would lead to a non-matching "route" which is still required for response validations. In this case the response validation will fail (if not ignored, too).
```

## Examples

* [Backend Validation](https://github.com/avenga/couper-examples/blob/master/backend-validation/README.md)

-----

## Navigation

* &#8673; [Blocks](../blocks.md)
* &#8672; [OIDC Block](beta-oidc.md) (Beta)
* &#8674; [Proxy Block](proxy.md)
