# `backend_responses`

`backend_responses.<label>` is a list of all backend responses and their variables.
To access a specific request use the related `label`. [Request Block](../blocks/request.md)
and [Proxy Block](../blocks/proxy.md) without a label will be available as `default`.

To access the HTTP status code of the `default` response use `backend_responses.default.status`.

| Variable                 | Type    | Description |
| ------------------------ | ------- | ----------- |
| `body`                   | string  | Response message body. |
| `context.<label>.<name>` | string  | Context information from an [Access Control](../access-control.md). See [`request.context`](request.md#requestcontext). |
| `cookies.<name>`         | string  | Value of the `Cookie` request HTTP header field referenced by `<name>`. |
| `headers.<name>`         | string  | Value of the request HTTP header field referenced by lowercased `<name>`. |
| `json_body.<name>`       | string  | JSON decoded request message body. Media type must be `application/json` or `application/*+json`. |
| `status`                 | integer | HTTP status code. |

-----

## Navigation

* &#8673; [Variables](../variables.md)
* &#8672; [`backend_requests`](backend-requests.md)
* &#8674; [`couper`](couper.md)
