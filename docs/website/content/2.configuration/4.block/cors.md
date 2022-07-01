# Cors

The `cors` block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

<!--TODO: check if this information is correct -->
&#9888; Overrides the CORS behavior of the parent block.

| Block name | Context                                                                                                       | Label    | Nested block(s) |
|:-----------|:--------------------------------------------------------------------------------------------------------------|:---------|:----------------|
| `cors`     | [Server Block](#server-block), [Files Block](#files-block), [SPA Block](#spa-block), [API Block](#api-block). | no label | -               |

| Attribute(s)        | Type                     | Default | Description                                                                                                                                                                           | Characteristic(s)                                                                                                         | Example                                                                         |
|:--------------------|:-------------------------|:--------|:--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------------|
| `allowed_origins`   | string or tuple (string) | -       | An allowed origin or a list of allowed origins.                                                                                                                                       | Can be either of: a string with a single specific origin, `"*"` (all origins are allowed) or an array of specific origins | `allowed_origins = ["https://www.example.com", "https://www.another.host.org"]` |
| `allow_credentials` | bool                     | `false` | Set to `true` if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` HTTP header fields).                                                   | -                                                                                                                         | -                                                                               |
| `disable`           | bool                     | `false` | Set to `true` to disable the inheritance of CORS from the [Server Block](#server-block) in [Files Block](#files-block), [SPA Block](#spa-block) and [API Block](#api-block) contexts. | -                                                                                                                         | -                                                                               |
| `max_age`           | [duration](#duration)    | -       | Indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response HTTP header fields.                                     | &#9888; Can be cached                                                                                                     | `max_age = "1h"`                                                                |

**Note:** `Access-Control-Allow-Methods` is only sent in response to a CORS preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method (see the `allowed_method` attribute for [`api`](#api-block) or [`endpoint`](#endpoint-block) blocks).
