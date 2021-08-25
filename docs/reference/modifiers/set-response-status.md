# set_response_status

The [`set_response_status` Attribute](../attributes.md) allows to modify the response
HTTP status code to the given value.

If the HTTP status code ist set to `204`, the reponse body and the HTTP header
field `Content-Length` is removed from the client response and a warning is logged.

## Contexts

* [Backend Block](../blocks/backend.md)
* [Endpoint Block](../blocks/endpoint.md)
* [Error Handler Block](../blocks/error-handler.md)

## Example

```hcl
server "example" {
  endpoint "/" {
    response {
      set_response_status = 204
    }
  }
}
```

-----

## Navigation

* &#8673; [Modifiers](../modifiers.md)
* &#8672; [`set_response_headers`](set-response-headers.md)
* &#8674; [`add_request_headers`](add-request-headers.md)
