# Error Handler

The `error_handler` block lets you configure the handling of errors thrown in components configured by the parent blocks.

The error handler label specifies which [error type](/configuration/error-handling#error-types) should be handled. Multiple labels are allowed. The label can be omitted to catch all relevant errors. This has the same behavior as the error type `*`, that catches all errors explicitly.

Concerning child blocks and attributes, the `error_handler` block is similar to an [Endpoint Block](endpoint).

| Block name  |Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
| `error_handler` | [API Block](api), [Endpoint Block](endpoint), [Basic Auth Block](basic_auth), [JWT Block](jwt), [OAuth2 AC Block (Beta)](oauth2), [OIDC Block](oidc), [SAML Block](saml) | optional | [Proxy Block(s)](proxy),  [Request Block(s)](request), [Response Block](response), [Error Handler Block(s)](error_handler) |

| Attribute(s)            | Type             | Default | Description                                                                                                       | Characteristic(s)                                                                                                                                                                                                                                                                                                                                                                                                                               | Example                                                              |
|:------------------------|:-----------------|:--------|:------------------------------------------------------------------------------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------|
| `custom_log_fields`     | object           | -       | Defines log fields for [Custom Logging](/observation/logging).                                                  | &#9888; Inherited by nested blocks.                                                                                                                                                                                                                                                                                                                                                                                                             | -                                                                    |
| [Modifiers](/configuration/modifiers) | -                | -       | -                                                                                                                 | -                                                                                                                                                                                                                                                                                                                                                                                                                                               | -                                                                    |

Examples:

- [Error Handling for Access Controls](https://github.com/avenga/couper-examples/blob/master/error-handling-ba/README.md).
