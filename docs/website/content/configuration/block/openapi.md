---
title: 'OpenAPI'
slug: 'openapi'
---

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


{{< attributes >}}
[
  {
    "default": "",
    "description": "OpenAPI YAML definition file.",
    "name": "file",
    "type": "string"
  },
  {
    "default": "false",
    "description": "Logs request validation results, skips error handling.",
    "name": "ignore_request_violations",
    "type": "bool"
  },
  {
    "default": "false",
    "description": "Logs response validation results, skips error handling.",
    "name": "ignore_response_violations",
    "type": "bool"
  }
]
{{< /attributes >}}

### Empty query parameter values

A query parameter that is present without a value (`?q` or `?q=`) is validated as
an empty string. For a parameter with `type: string` an empty string satisfies both
`required: true` and the default `allowEmptyValue: false`, so such a request passes
request validation. A parameter of any other type — `integer`, for example — is
still rejected, and a parameter that is absent altogether still fails the
`required` check.

Use `minLength: 1` in the parameter schema to reject empty values:

```yaml
parameters:
  - in: query
    name: q
    required: true
    schema:
      type: string
      minLength: 1
```

### Example

```hcl
openapi {
  file = "openapi.yaml"
  ignore_response_violations = true
}
```

You can find a detailed example [here](https://github.com/coupergateway/couper-examples/blob/master/backend-validation/README.md).
