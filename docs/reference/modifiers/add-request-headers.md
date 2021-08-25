# add_request_headers

The [`add_request_headers` Attribute](../attributes.md) allows to add key/value(s)
pairs to the HTTP header fields of the upstream request.

## Execution order of `*_request_headers`

1. [`remove_request_headers`](remove-request-headers.md)
2. [`set_request_headers`](set-request-headers.md)
3. `add_request_headers`

## Contexts

* [Backend Block](../blocks/backend.md)
* [Endpoint Block](../blocks/endpoint.md)
* [Error Handler Block](../blocks/error-handler.md)
* [Proxy Block](../blocks/proxy.md)

## Example

```hcl
server "example" {
  endpoint "/" {
    proxy {
      url = "https://example.com"

      add_request_headers = {
        Cache-Control = "no-cache"
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Modifiers](../modifiers.md)
* &#8672; [`set_response_status`](set-response-status.md)
* &#8674; [`add_response_headers`](add-response-headers.md)
