# set_response_headers

The [`set_response_headers` Attribute](../attributes.md) allows to set key/value(s)
pairs to the HTTP header fields of the client response.

## Execution order of `*_response_headers`

1. [`remove_response_headers`](remove-response-headers.md)
2. `set_response_headers`
3. [`add_response_headers`](add-response-headers.md)

## Contexts

* [API Block](../blocks/api.md)
* [Backend Block](../blocks/backend.md)
* [Endpoint Block](../blocks/endpoint.md)
* [Error Handler Block](../blocks/error-handler.md)
* [Files Block](../blocks/files.md)
* [Proxy Block](../blocks/proxy.md)
* [Response Block](../blocks/response.md)
* [Server Block](../blocks/server.md)
* [SPA Block](../blocks/spa.md)

## Example

```hcl
server "example" {
  endpoint "/" {
    proxy {
      url = "https://example.com"

      set_response_headers = {
        Cache-Control = "public"
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Modifiers](../modifiers.md)
* &#8672; [`set_request_headers`](set-request-headers.md)
* &#8674; [`set_response_status`](set-response-status.md)
