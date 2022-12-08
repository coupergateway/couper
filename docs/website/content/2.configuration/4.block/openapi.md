# OpenAPI

The `openapi` block configures the backend's proxy behavior to validate outgoing
and incoming requests to and from the origin, preventing the origin from invalid
requests and the Couper client from invalid answers.
To do so Couper uses the [OpenAPI 3 standard](https://www.openapis.org/) to load
the definitions from a given document defined with the `file` attribute.

⚠️ While ignoring request violations an invalid method or path would
lead to a non-matching _route_ which is still required for response validations.
In this case the response validation will fail if not ignored, too.

| Block name | Context                                       | Label    |
|:-----------|:----------------------------------------------|:---------|
|`openapi`   | [Backend Block](/configuration/block/backend) | no label |


::attributes
---
values: [
  {
    "default": "",
    "description": "OpenAPI YAML definition file",
    "name": "file",
    "type": "string"
  },
  {
    "default": "false",
    "description": "logs request validation results, skips error handling",
    "name": "ignore_request_violations",
    "type": "bool"
  },
  {
    "default": "false",
    "description": "logs response validation results, skips error handling",
    "name": "ignore_response_violations",
    "type": "bool"
  }
]

---
::

### Example

```hcl
openapi {
  file = "openapi.yaml"
  ignore_response_violations = true
}
```

You can find a detailed example [here](https://github.com/avenga/couper-examples/blob/master/backend-validation/README.md).
