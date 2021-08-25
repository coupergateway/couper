# add_form_params

The [`add_form_params` Attribute](../attributes.md) allows to add form parameters
as key/value(s) pairs to the upstream request body.

## Execution order of `*_form_params`

1. [`remove_form_params`](remove-form-params.md)
2. [`set_form_params`](set-form-params.md)
3. `add_form_params`

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

      add_form_params = {
        sort = "desc"
      }
    }
  }
}
```

-----

## Navigation

* &#8673; [Parameters](../parameters.md)
* &#8672; [`set_query_params`](set-query-params.md)
* &#8674; [`add_query_params`](add-query-params.md)
