# `backend_requests`

The `backend_requests.<label>` is a list of all backend requests and their variables.
To access a specific request use the related `label`. [Request Block](../blocks/request.md)
and [Proxy Block](../blocks/proxy.md) without a label will be available as `default`.

To access the HTTP method of the `default` request use `backend_requests.default.method`.

| Variable                 | Type                            | Description |
| ------------------------ | ------------------------------- | ----------- |
| `body`                   | string                          | Request message body. |
| `context.<label>.<name>` | string                          | Context information from an [Access Control](../access-control.md). See [`request.context`](request.md#requestcontext). |
| `cookies.<name>`         | string                          | Value of the `Cookie` request HTTP header field referenced by `<name>`. |
| `form_body.<name>`       | [list](../config-types.md#list) | Parameter from an `application/x-www-form-urlencoded` body. |
| `headers.<name>`         | string                          | Value of the request HTTP header field referenced by lowercased `<name>`. |
| `host`                   | string                          | Host part of the request URL. |
| `id`                     | string                          | Unique request ID. |
| `json_body.<name>`       | string                          | JSON decoded request message body. Media type must be `application/json` or `application/*+json`. |
| `method`                 | string                          | HTTP request method. |
| `origin`                 | string                          | Origin part of the request URL. |
| `path`                   | string                          | Path part of the request URL. |
| `port`                   | integer                         | Port part of the request URL. |
| `protocol`               | string                          | Request protocol. |
| `query.<name>`           | [list](../config-types.md#list) | Query parameter values from the request URL referenced by `<name>`. |
| `url`                    | string                          | Request URL. |

-----

## Navigation

* &#8673; [Variables](../variables.md)
* &#8672; [`request`](request.md)
* &#8674; [`backend_responses`](backend-responses.md)
