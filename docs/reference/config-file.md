# Configuration Reference ~ Configuration File

The language for Couper's configuration file is
[HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax),
a configuration language by HashiCorp.

## File Name

The file-ending of Your configuration file should be `.hcl` to have syntax highlighting
within Your IDE.

The file name defaults to `couper.hcl` in the working directory. This can be changed
with the `-f` [Command Line](cli.md#global-options) flag. With `-f /opt/couper/my_conf.hcl`
couper changes the working directory to `/opt/couper` directory and loads `my_conf.hcl`
file.

## IDE Extension

Couper provides its own IDE extension that adds Couper-specific highlighting and
autocompletion to Couper's configuration file `couper.hcl` in Visual Studio Code.

Get it from the [Visual Studio Market Place](https://marketplace.visualstudio.com/items?itemName=AvengaGermanyGmbH.couper)
or visit the [Extension repository](https://github.com/avenga/couper-vscode).

## Basic File Structure

Couper's configuration file consists of nested configuration [Blocks](blocks.md) that
configure the gateway. The main structure:

```hcl
server "example" {
  # ...
}

definitions {
  # ...
}

settings {
  # ...
}

defaults {
  # ...
}
```

## Expressions

Since Couper's configuration file uses [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax)
for the configuration, it is able to use attribute values as expression.

```hcl
// Arithmetic with literals and application-provided variables
sum = 1 + addend

// String interpolation and templates
message = "Hello, ${name}!"

// Application-provided functions
shouty_message = to_upper(message)
```

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Command Line Interface](cli.md)
* &#8674; [Configuration Types](config-types.md)
