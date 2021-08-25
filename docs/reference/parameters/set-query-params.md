# set_query_params

The [`set_query_params` Attribute](../attributes.md) allows to set query parameters
as key/value(s) pairs in the upstream request URL.

## Execution order of `*_query_params`

1. [`remove_query_params`](remove-query-params.md)
2. `set_query_params`
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

      set_query_params = {
        key = "value"
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Parameters](../parameters.md)
* &#8672; [`set_form_params`](set-form-params.md)
* &#8674; [`add_form_params`](add-form-params.md)
