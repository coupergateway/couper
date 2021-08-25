# add_response_headers

The [`add_response_headers` Attribute](../attributes.md) allows to add key/value(s)
pairs to the HTTP header fields of the client response.

## Execution order of `*_response_headers`

1. [`remove_response_headers`](remove-response-headers.md)
2. [`set_response_headers`](set-response-headers.md)
3. `add_response_headers`

## Contexts

* [API Block](../blocks/api.md)
* [Backend Block](../blocks/backend.md)
* [Endpoint Block](../blocks/endpoint.md)
* [Error Handler Block](../blocks/error-handler.md)
* [Files Block](../blocks/files.md)
* [Proxy Block](../blocks/proxy.md)
* [Server Block](../blocks/server.md)
* [SPA Block](../blocks/spa.md)

## Example

```hcl
server "example" {
  endpoint "/" {
    proxy {
      url = "https://example.com"

      add_response_headers = {
        Cache-Control = "public"
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Modifiers](../modifiers.md)
* &#8672; [`add_request_headers`](add-request-headers.md)
* &#8674; [`remove_request_headers`](remove-request-headers.md)
