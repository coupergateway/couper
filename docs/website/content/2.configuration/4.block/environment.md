# Environment Block

The `environment` block lets you refine the Couper configuration based on the set
[environment](/configuration/command-linemd#global-options).

| Block name    | Context  | Label                                            | Nested block(s)                     |
| :------------ | :------- | :----------------------------------------------- | :---------------------------------- |
| `environment` | Overall. | &#9888; required, multiple labels are supported. | All configuration blocks of Couper. |

The `environment` block works like a preprocessor. If the label of an `environment`
block do not match the set [environment](./CLI.md#global-options) value, the preprocessor
removes this block and their content. Otherwise, the content of the block is applied
to the configuration.

If the [environment](/configuration/command-linemd#global-options) value set to `prod`, the following configuration

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

produces after the preprocessing the following configuration:

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
