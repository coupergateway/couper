# Websockets

The `websockets` block activates support for websocket connections in Couper.

| Block name   | Context                     | Label    | Nested block(s) |
|:-------------|:----------------------------|:---------|:----------------|
| `websockets` | [Proxy Block](#proxy-block) | no label | -               |

| Attribute(s)          | Type                  | Default | Description                                                       | Characteristic(s)                                                   | Example          |
|:----------------------|:----------------------|:--------|:------------------------------------------------------------------|:--------------------------------------------------------------------|:-----------------|
| `timeout`             | [duration](#duration) | -       | The total deadline duration a websocket connection has to exists. | -                                                                   | `timeout = 600s` |
| `set_request_headers` | -                     | -       | -                                                                 | Same as `set_request_headers` in [Request Header](#request-header). | -                |
