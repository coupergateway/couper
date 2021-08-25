# remove_form_params

The [`remove_form_params` Attribute](../attributes.md) allows to remove form parameters
from the upstream request body.

## Execution order of `*_form_params`

1. `remove_form_params`
2. [`set_form_params`](set-form-params.md)
3. [`add_form_params`](add-form-params.md)

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

      remove_form_params = ["sort"]
    }
  }
}
```

-----

## Navigation

* &#8673; [Parameters](../parameters.md)
* &#8672; [`add_query_params`](add-query-params.md)
* &#8674; [`remove_query_params`](remove-query-params.md)
