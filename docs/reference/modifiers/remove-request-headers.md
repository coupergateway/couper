# remove_request_headers

The [`remove_request_headers` Attribute](../attributes.md) allows to remove HTTP
header fields from the upstream request.

## Execution order of `*_request_headers`

1. `remove_request_headers`
2. [`set_request_headers`](set-request-headers.md)
3. [`add_request_headers`](add-request-headers.md)

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

      remove_request_headers = ["Cache-Control"]
    }
  }
}
```

-----

## Navigation

* &#8673; [Modifiers](../modifiers.md)
* &#8672; [`add_response_headers`](add-response-headers.md)
* &#8674; [`remove_response_headers`](remove-response-headers.md)
