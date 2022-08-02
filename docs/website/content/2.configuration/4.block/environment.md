# Environment

The `environment` block lets you refine the Couper configuration based on the set
[environment](../command-line#global-options).

| Block name    | Context  | Label                                            | Nested block(s)                     |
| :------------ | :------- | :----------------------------------------------- | :---------------------------------- |
| `environment` | Overall. | &#9888; required, multiple labels are supported. | All configuration blocks of Couper. |

The `environment` block works like a preprocessor. If the label of an `environment`
block does not match the set [`COUPER_ENVIRONMENT`](../command-line#global-options) value, the preprocessor
removes this block and its content. Otherwise, the content of the block is added
to the configuration.

## Example

Considering the following configuration with the `COUPER_ENVIRONMENT` value set to `prod`

```hcl
server {
  api "protected" {
    endpoint "/secure" {
      environment "prod" {
        access_control = ["jwt"]
      }

      proxy {
        environment "prod" {
          url = "https://protected-resource.org"
        }
        environment "stage" {
          url = "https://test-resource.org"
        }
      }
    }
  }
}
```

the result will be:

```hcl
server {
  api "protected" {
    endpoint "/secure" {
      access_control = ["jwt"]

      proxy {
        url = "https://protected-resource.org"
      }
    }
  }
}
```
