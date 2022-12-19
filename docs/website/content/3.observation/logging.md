# Logging

## Introduction

Upon the execution of Couper all log events are sent to the standard output. You can adjust the appearance and verbosity of the logs with a variety of [Settings](/observation/logging#settings). There are different [Log Types](#log-types) and [Fields](#fields) containing useful information.

> We aspire to make Couper as stable as possible so these logs are still subject to change.

## Log Types

| Type                | Description                                                                                                                                                                                                |
|:--------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `couper_access`     | Provides information about the frontend side of things. Compare [Access Fields](#access-fields).                                                                                                           |
| `couper_access_tls` | Provides information about connections via configured [https_dev_proxy](/configuration/block/settings). Compare [Access Fields](#access-fields).                                                           |
| `couper_backend`    | Provides information about the backend side of things. Compare [Backend Fields](#backend-fields).                                                                                                          |
| `couper_daemon`     | Provides background information about the execution of Couper. Each printed log of this type contains a `message` entry describing the current actions of Couper. Compare [Daemon Fields](#daemon-fields). |
| `couper_job`        | Provides information about [jobs](/configuration/block/job). See [Job Fields](#job-fields).                                                                                                                |

## Fields

Given the large amount of information contained in some logs, it might come handy to change the format or log level (see [Settings](/configuration/block/settings)).

### Common Fields

These fields are part of all [Log Types](#log-types) and therefore mentioned separately to avoid unnecessary redundancy.

| Name          | Description                                       |
|:--------------|:--------------------------------------------------|
| `"build”`     | git short hash during build time                  |
| `"level"`     | configured log level, determines verbosity of log |
| `"message”`   | context based, mainly used in `couper_daemon`     |
| `"timestamp"` | request starting time                             |
| `"type"`      | [Log Type](#log-types)                            |
| `"version"`   | release version                                   |

### Access Fields

These fields are found in the [Log Types](#log-types) `couper_access` and `couper_access_tls` in addition to the [Common Fields](#common-fields).

| Name          |             | Description                                                                                                                                                                                                          |
|:--------------|:------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `"auth_user"` |             | basic auth username (if provided)                                                                                                                                                                                    |
| `"client_ip"` |             | ip of client                                                                                                                                                                                                         |
| `"custom"`    |             | see [Custom Logging](#custom-logging)                                                                                                                                                                                |
| `"endpoint"`  |             | path pattern of endpoint                                                                                                                                                                                             |
| `"handler"`   |             | one of: `endpoint`, `file`, `spa`                                                                                                                                                                                    |
| `"method"`    |             | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information                                                                                    |
| `"port"`      |             | current port accepting request                                                                                                                                                                                       |
| `"request":`  |             | field regarding request information                                                                                                                                                                                  |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"bytes"`   | request body size in bytes                                                                                                                                                                                           |
|               | `"headers"` | field regarding keys and values originating from configured keys/header names                                                                                                                                        |
|               | `"host"`    | request host                                                                                                                                                                                                         |
|               | `"method"`  | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information                                                                                    |
|               | `"origin"`  | request origin (`<proto>://<host>:<port>`), for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information |
|               | `"path"`    | request path                                                                                                                                                                                                         |
|               | `"proto"`   | request protocol                                                                                                                                                                                                     |
|               | `"status"`  | request status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information                                                                                     |
|               | `"tls"`     | TLS used `true` or `false`                                                                                                                                                                                           |
|               | `}`         |                                                                                                                                                                                                                      |
| `"response":` |             | field regarding response information                                                                                                                                                                                 |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"bytes"`   | response body size in bytes                                                                                                                                                                                          |
|               | `"headers"` | field regarding keys and values originating from configured keys/header names                                                                                                                                        |
|               | `}`         |                                                                                                                                                                                                                      |
| `"server"`    |             | server name (defined in couper file)                                                                                                                                                                                 |
| `"status"`    |             | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information                                                                                    |
| `"timings":`  |             | field regarding timing [ms]                                                                                                                                                                                          |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"total"`   | total time taken                                                                                                                                                                                                     |
|               | `}`         |                                                                                                                                                                                                                      |
| `“uid"`       |             | unique request id configurable in [Settings](/configuration/block/settings)                                                                                                                                          |
| `"url"`       |             | complete url (`<proto>://<host>:<port><path>` or `<origin><path>`)                                                                                                                                                   |

### Backend Fields

These fields are found in the [Log Type](#log-types) `couper_backend` in addition to the [Common Fields](#common-fields).

| Name                    |             | Description                                                                                                                                                                              |
|:------------------------|:------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `"auth_user"`           |             | backend request basic auth username (if provided)                                                                                                                                        |
| `"backend"`             |             | configured name, `default` if not provided                                                                                                                                               |
| `"custom"`              |             | see [Custom Logging](#custom-logging)                                                                                                                                                    |
| `"method"`              |             | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information                                                        |
| `"proxy"`               |             | used system proxy url (if configured), see [Proxy Block](/configuration/block/proxy)                                                                                                     |
| `"request":`            |             | field regarding request information                                                                                                                                                      |
|                         | `{`         |                                                                                                                                                                                          |
|                         | `"bytes"`   | request body size in bytes                                                                                                                                                               |
|                         | `"headers"` | field regarding keys and values originating from configured keys/header names                                                                                                            |
|                         | `"host"`    | request host                                                                                                                                                                             |
|                         | `"method"`  | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information                                                        |
|                         | `"name"`    | configured request name, `default` if not provided                                                                                                                                       |
|                         | `"origin"`  | request origin, for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information |
|                         | `"path"`    | request path                                                                                                                                                                             |
|                         | `"port"`    | current port accepting request                                                                                                                                                           |
|                         | `"proto"`   | request protocol                                                                                                                                                                         |
|                         | `}`         |                                                                                                                                                                                          |
| `"response":`           |             | field regarding response information                                                                                                                                                     |
|                         | `{`         |                                                                                                                                                                                          |
|                         | `"bytes"`   | raw size of read body bytes                                                                                                                                                              |
|                         | `"headers"` | field regarding keys and values originating from configured keys/header names                                                                                                            |
|                         | `"status"`  | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information                                                        |
|                         | `}`         |                                                                                                                                                                                          |
| `"status"`              |             | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information                                                        |
| `"timings":`            |             | field regarding timing [ms]                                                                                                                                                              |
|                         | `{`         |                                                                                                                                                                                          |
|                         | `"dns"`     | time taken by dns                                                                                                                                                                        |
|                         | `"tcp"`     | time taken between attempting and establishing tcp connection                                                                                                                            |
|                         | `"tls"`     | time taken between attempt and success at tls handshake                                                                                                                                  |
|                         | `"total"`   | total time taken                                                                                                                                                                         |
|                         | `"ttfb"`    | time to first byte/between establishing connection and receiving first byte                                                                                                              |
|                         | `}`         |                                                                                                                                                                                          |
| `"token_request"`       |             | entry regarding request for token                                                                                                                                                        |
| `"token_request_retry"` |             | how many `token_request` attempts were made                                                                                                                                              |
| `"uid"`                 |             | unique request id configurable in [Settings](/configuration/block/settings)                                                                                                              |
| `"url"`                 |             | complete url (`<proto>://<host>:<port><path>` or `<origin><path>`)                                                                                                                       |
| `"validation"`          |             | validation result for open api, see [OpenAPI Block](/configuration/block/open-api)                                                                                                        |

### Daemon Fields

These fields are found in the [Log Type](#log-types) `couper_daemon` in addition to the [Common Fields](#common-fields).

| Name         |                 | Description                                                                                                                |
|:-------------|:----------------|:---------------------------------------------------------------------------------------------------------------------------|
| `"deadline"` |                 | shutdown parameter, see [Health Check](/observation/health)                                                        |
| `"delay"`    |                 | shutdown parameter, see [Health Check](/observation/health)                                                        |
| `"watch":`   |                 | field watching configuration file changes, logs with this field only appear if `watch=true`, more in [Settings](/configuration/block/settings) |
|              | `{`             |                                                                                                                            |
|              | `"max-retries"` | maximum retry count, see [Global Options](/configuration/command-line#global-options)                                                         |
|              | `"retry-delay"` | configured delay of each retry, see [Global Options](/configuration/command-line#global-options)                                              |
|              | `}`             |                                                                                                                            |

## Job Fields

The following fields are found in the [log type](#log-types) `couper_daemon` in addition to the [common fields](#common-fields).

| Name         |              | Description                                                                    |
|:-------------|:-------------|:-------------------------------------------------------------------------------|
| `"name"`     |              | job name, label of [`beta_job` block](/configuration/block/job)                |
| `"timings":` |              | field regarding timing [ms]                                                    |
|              | `{`          |                                                                                |
|              | `"interval"` | interval, see [`interval` attribute](/configuration/block/job#attributes)      |
|              | `"total"`    | total time taken                                                               |
|              | `}`          |                                                                                |
| `“uid"`      |              | unique request id configurable in [settings](/configuration/block/settings)    |

## Custom Logging

These fields are defined in the configuration as `custom_log_fields` attribute in the following blocks:

- [Server Block](/configuration/block/server)
- [Files Block](/configuration/block/files)
- [SPA Block](/configuration/block/spa)
- [API Block](/configuration/block/api)
- [Endpoint Block](/configuration/block/endpoint)
- [Backend Block](/configuration/block/backend)
- [Basic Auth Block](/configuration/block/basic_auth)
- [JWT Block](/configuration/block/jwt)
- [OAuth2 AC (Beta) Block](/configuration/block/beta_oauth2)
- [OIDC Block](/configuration/block/oidc)
- [SAML Block](/configuration/block/saml)
- [Error Handler Block](/configuration/error-handling)

All `custom_log_fields` definitions will take place within the `couper_access` log with the `custom` field as parent.
Except the `custom_log_fields` defined in a [`backend` block](/configuration/block/backend) which will take place
in the `couper_backend` log.

**Example:**

```hcl
server "example" {
  endpoint "/anything" {
    custom_log_fields = {
      origin  = backend_responses.default.json_body.origin
      success = backend_responses.default.status == 200
    }

    proxy {
      backend = "httpbin"
      path = "/anything"
    }
  }
}

definitions {
  backend "httpbin" {
    origin = "https://httpbin.org"

    custom_log_fields = {
      method = request.method
    }
  }
}
```

## Settings

For more information regarding the usage of these settings, see the [Command Line Interface](/configuration/command-line) documentation and the [`settings` block  reference](/configuration/block/settings).

| Feature      | Description                                                                                                                                                                                                                                    |
|:-------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `log-format` | Can be set to either `common` or `json` output format.                                                                                                                                                                                         |
| `log-level`  | Can be set to one of: `panic`, `fatal`, `error`, `warn`, `info`, `debug` and `trace`. For more information regarding the usage of each, see the official documentation of [Level](https://pkg.go.dev/github.com/sirupsen/logrus@v1.8.1#Level). |
| `log-pretty` | Option for `log-format=json` which pretty prints with basic key coloring.                                                                                                                                                                      |
| `watch`      | Watch for configuration file changes and reload on modifications. Resulting information manifests in [Daemon Logs](#daemon-fields).                                                                                                            |
