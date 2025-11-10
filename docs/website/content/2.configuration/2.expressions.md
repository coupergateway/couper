---
title: 'Expressions'
description: 'Basic explanation of how to use hcl expressions.'
---

# Expressions

Since we use [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax) for our configuration, we are able to use attribute values as expression.

```hcl
// Arithmetic with literals and application-provided variables.
sum = 1 + addend

// String interpolation and templates.
message = "Hello, ${name}!"

// Application-provided functions.
shouty_message = upper(message)
```

See [functions](/configuration/functions).
