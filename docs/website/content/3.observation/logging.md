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

| Name          | Description                                        |
|:--------------|:---------------------------------------------------|
| `"build”`     | GIT short hash during build time.                  |
| `"level"`     | Configured log level, determines verbosity of log. |
| `"message”`   | Context based, mainly used in `couper_daemon`.     |
| `"timestamp"` | Request starting time.                             |
| `"type"`      | [Log Type](#log-types).                            |
| `"version"`   | Release version.                                   |

### Access Fields

These fields are found in the [Log Types](#log-types) `couper_access` and `couper_access_tls` in addition to the [Common Fields](#common-fields).

| Name          |             | Description                                                                                                                                                                                                          |
|:--------------|:------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `"auth_user"` |             | Basic auth username (if provided).                                                                                                                                                                                    |
| `"client_ip"` |             | IP of client.                                                                                                                                                                                                         |
| `"custom"`    |             | See [Custom Logging](#custom-logging).                                                                                                                                                                                |
| `"endpoint"`  |             | Path pattern of endpoint.                                                                                                                                                                                             |
| `"handler"`   |             | One of: `endpoint`, `file`, `spa`.                                                                                                                                                                                    |
| `"method"`    |             | HTTP request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information.                                                                                    |
| `"port"`      |             | Current port accepting request.                                                                                                                                                                                       |
| `"request":`  |             | Field regarding request information.                                                                                                                                                                                  |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"bytes"`   | Request body size in bytes.                                                                                                                                                                                           |
|               | `"headers"` | Field regarding keys and values originating from configured keys/header names.                                                                                                                                        |
|               | `"host"`    | Request host.                                                                                                                                                                                                         |
|               | `"method"`  | HTTP request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information.                                                                                    |
|               | `"origin"`  | Request origin (`<proto>://<host>:<port>`), for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information. |
|               | `"path"`    | Request path.                                                                                                                                                                                                         |
|               | `"proto"`   | Request protocol.                                                                                                                                                                                                     |
|               | `"status"`  | Request status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information.                                                                                     |
|               | `"tls"`     | TLS used `true` or `false`.                                                                                                                                                                                           |
|               | `}`         |                                                                                                                                                                                                                      |
| `"response":` |             | Field regarding response information.                                                                                                                                                                                 |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"bytes"`   | Response body size in bytes.                                                                                                                                                                                          |
|               | `"headers"` | Field regarding keys and values originating from configured keys/header names.                                                                                                                                        |
|               | `}`         |                                                                                                                                                                                                                      |
| `"server"`    |             | Server name (if defined in couper file).                                                                                                                                                                                 |
| `"status"`    |             | Response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information.                                                                                    |
| `"timings":`  |             | Field regarding timing (ms).                                                                                                                                                                                          |
|               | `{`         |                                                                                                                                                                                                                      |
|               | `"total"`   | Total time taken.                                                                                                                                                                                                     |
|               | `}`         |                                                                                                                                                                                                                      |
| `“uid"`       |             | Unique request ID configurable in [Settings](/configuration/block/settings).                                                                                                                                          |
| `"url"`       |             | Complete URL (`<proto>://<host>:<port><path>` or `<origin><path>`).                                                                                                                                                   |

### Backend Fields

These fields are found in the [Log Type](#log-types) `couper_backend` in addition to the [Common Fields](#common-fields).

| Name                    |             | Description                                                                                                                                                                               |
|:------------------------|:------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `"auth_user"`           |             | Backend request basic auth username (if provided).                                                                                                                                        |
| `"backend"`             |             | Configured name (`default` if not provided).                                                                                                                                              |
| `"custom"`              |             | See [Custom Logging](#custom-logging).                                                                                                                                                    |
| `"method"`              |             | HTTP request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information.                                                        |
| `"proxy"`               |             | Used system proxy URL (if configured), see [Proxy Block](/configuration/block/proxy).                                                                                                     |
| `"request":`            |             | Field regarding request information.                                                                                                                                                      |
|                         | `{`         |                                                                                                                                                                                           |
|                         | `"bytes"`   | Request body size in bytes.                                                                                                                                                               |
|                         | `"headers"` | Field regarding keys and values originating from configured keys/header names.                                                                                                            |
|                         | `"host"`    | Request host.                                                                                                                                                                             |
|                         | `"method"`  | HTTP request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information.                                                        |
|                         | `"name"`    | Configured request name (`default` if not provided).                                                                                                                                      |
|                         | `"origin"`  | Request origin, for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information. |
|                         | `"path"`    | Request path.                                                                                                                                                                             |
|                         | `"port"`    | Current port accepting request.                                                                                                                                                           |
|                         | `"proto"`   | Request protocol.                                                                                                                                                                         |
|                         | `}`         |                                                                                                                                                                                           |
| `"response":`           |             | Field regarding response information.                                                                                                                                                     |
|                         | `{`         |                                                                                                                                                                                           |
|                         | `"bytes"`   | Raw size of read body bytes.                                                                                                                                                              |
|                         | `"headers"` | Field regarding keys and values originating from configured keys/header names.                                                                                                            |
|                         | `"status"`  | Response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information.                                                        |
|                         | `}`         |                                                                                                                                                                                           |
| `"status"`              |             | Response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information.                                                        |
| `"timings":`            |             | Field regarding timing (ms).                                                                                                                                                              |
|                         | `{`         |                                                                                                                                                                                           |
|                         | `"dns"`     | Time taken by DNS.                                                                                                                                                                        |
|                         | `"tcp"`     | Time taken between attempting and establishing TCP connection.                                                                                                                            |
|                         | `"tls"`     | Time taken between attempt and success at TLS handshake.                                                                                                                                  |
|                         | `"total"`   | Total time taken.                                                                                                                                                                         |
|                         | `"ttfb"`    | Time to first byte/between establishing connection and receiving first byte.                                                                                                              |
|                         | `}`         |                                                                                                                                                                                           |
| `"token_request"`       |             | Entry regarding request for token.                                                                                                                                                        |
| `"token_request_retry"` |             | How many `token_request` attempts were made.                                                                                                                                              |
| `"uid"`                 |             | Unique request ID configurable in [Settings](/configuration/block/settings)                                                                                                               |
| `"url"`                 |             | Complete URL (`<proto>://<host>:<port><path>` or `<origin><path>`).                                                                                                                       |
| `"validation"`          |             | Validation result for open api, see [OpenAPI Block](/configuration/block/openapi).                                                                                                        |

### Daemon Fields

These fields are found in the [Log Type](#log-types) `couper_daemon` in addition to the [Common Fields](#common-fields).

| Name         |                 | Description                                                                                                                                    |
|:-------------|:----------------|:-----------------------------------------------------------------------------------------------------------------------------------------------|
| `"deadline"` |                 | Shutdown parameter, see [Health Check](/observation/health).                                                                                    |
| `"delay"`    |                 | Shutdown parameter, see [Health Check](/observation/health).                                                                                    |
| `"watch":`   |                 | Field watching configuration file changes, logs with this field only appear if `watch=true`, more in [Settings](/configuration/block/settings). |
|              | `{`             |                                                                                                                                                |
|              | `"max-retries"` | Maximum retry count, see [Basic Options](/configuration/command-line#basic-options).                                                          |
|              | `"retry-delay"` | Configured delay of each retry, see [Basic Options](/configuration/command-line#basic-options).                                               |
|              | `}`             |                                                                                                                                                |

## Job Fields

The following fields are found in the [log type](#log-types) `couper_daemon` in addition to the [common fields](#common-fields).

| Name         |              | Description                                                                    |
|:-------------|:-------------|:-------------------------------------------------------------------------------|
| `"name"`     |              | Job name, label of [`job` block](/configuration/block/job) (`beta_job` alias). |
| `"timings":` |              | Field regarding timing (ms).                                                   |
|              | `{`          |                                                                                |
|              | `"interval"` | Interval, see [`interval` attribute](/configuration/block/job#attributes).     |
|              | `"total"`    | Total time taken.                                                              |
|              | `}`          |                                                                                |
| `“uid"`      |              | Unique request ID configurable in [settings](/configuration/block/settings).   |

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
