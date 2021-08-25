# remove_query_params

The [`remove_query_params` Attribute](../attributes.md) allows to remove query parameters
from the upstream request URL.

## Execution order of `*_query_params`

1. `remove_query_params`
2. [`set_query_params`](set-query-params.md)
3. [`add_query_params`](add-query-params.md)

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

      remove_query_params = ["key"]
    }
  }
}
```

-----

## Navigation

* &#8673; [Parameters](../parameters.md)
* &#8672; [`remove_form_params`](remove-form-params.md)
* &#8674; [`set_form_params`](set-form-params.md)
