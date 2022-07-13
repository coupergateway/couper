# OpenAPI

The `openapi` block configures the backends proxy behavior to validate outgoing
and incoming requests to and from the origin. Preventing the origin from invalid
requests, and the Couper client from invalid answers. An example can be found
[here](https://github.com/avenga/couper-examples/blob/master/backend-validation/README.md).
To do so Couper uses the [OpenAPI 3 standard](https://www.openapis.org/) to load
the definitions from a given document defined with the `file` attribute.

&#9888; While ignoring request violations an invalid method or path would
lead to a non-matching _route_ which is still required for response validations.
In this case the response validation will fail if not ignored too.

|Block name|Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
|`openapi`| [Backend Block](backend)|-|-|

| Attribute(s) | Type |Default|Description|Characteristic(s)| Example|
| :------------------------------ | :--------------- | :--------------- | :--------------- | :--------------- | :--------------- |
| `file`                       |string|-|OpenAPI yaml definition file.|&#9888; required|`file = "openapi.yaml"`|
| `ignore_request_violations`  |bool|`false`|Log request validation results, skip error handling. |-|-|
| `ignore_response_violations` |bool|`false`|Log response validation results, skip error handling.|-|-|
