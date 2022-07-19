# Response

The `response` block creates and sends a client response.

| Block name | Context                           | Label    | Nested block(s) |
|:-----------|:----------------------------------|:---------|:----------------|
| `response` | [Endpoint Block](endpoint) | no label | -               |

| Attribute(s) | Type                                      | Default | Description                                                           | Characteristic(s)                                                       | Example |
|:-------------|:------------------------------------------|:--------|:----------------------------------------------------------------------|:------------------------------------------------------------------------|:--------|
| `body`       | string                                    | -       | -                                                                     | Creates implicit default `Content-Type: text/plain` header field.       | -       |
| `json_body`  | null, bool, number, string, object, tuple | -       | -                                                                     | Creates implicit default `Content-Type: application/json` header field. | -       |
| `status`     | integer                                   | `200`   | HTTP status code.                                                     | -                                                                       | -       |
| `headers`    | object                                    | -       | Same as `set_response_headers` in [Request Header](../modifiers#response-header). | -                                                                       | -       |
