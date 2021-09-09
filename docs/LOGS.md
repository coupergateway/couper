- [Introduction](#introduction)
  - [Log Types](#log-types)
  - [Fields](#fields)
    - [Access Fields](#access-fields)
    - [Backend Fields](#backend-fields)
    - [Daemon Fields](#daemon-fields)
  - [Settings](#settings)

# Introduction

Upon the execution of Couper all log events are sent to the standard output. You can adjust the appearance and verbosity of the logs with a variety of [Settings](#settings). There are different [Log Types](#log-types) and [Fields](#fields) containing useful information.```

These _Logs_ are however still subject to change. We aspire to make Couper as stable as possible and that includes the _Logs_, so until we believe to have achieved that goal you should expect to see some changes.

## Log Types

| Type                | Description                                                                                                                                     |
| :------------------ | :---------------------------------------------------------------------------------------------------------------------------------------------- |
| `couper_access`     | Provides information about the frontend side of things. For information about its [Fields](#fields), see [Access Fields](#access-fields).       |
| `couper_access_tls` | Provides information about connections via configured [https_dev_proxy](./REFERENCE.md#settings). For information about its [Fields](#fields), see [Access Fields](#access-fields).                                                                                                                                        |
| `couper_backend`    | Provides information about the backend side of things. For information about its [Fields](#fields), see [Backend Fields](#backend-fields).      |
| `couper_daemon`     | Provides background information about the execution of Couper. It is here where the main purpose of the `message` [Field](#fields) lies, as each printed _Log_ of this type will contain a `message` entry containing a description of the current actions of Couper. For information about its [Fields](#fields), see [Daemon Fields](#daemon-fields).                                                                                                                                                       |

## Fields

Given the large amount of information contained in several _Logs_, see [Settings](#settings) to change the format or level of intricacy with which _Logs_ are printed.

### Common Fields

These are a part of all [Log Types](#log-types) and are hence mentioned seperately to avoid unnecessary redundancy.

| Name          | Description                                       |
| :------------ | :------------------------------------------------ |
| `"build”`     | git short hash during build time                  |
| `"level"`     | configured log level, determines verbosity of log |
| `"message”`   | context based, mainly used in `couper_daemon`     |
| `"timestamp"` | request starting time                             |
| `"type"`      | [Log Type](#log-types)                            |
| `"version"`   | release version                                   |

### Access Fields

These are found in the [Log Types](#log-types) `couper_access` and `couper_access_tls` in addition to the [Common Fields](#common-fields).

| Name                      | Description                                                                                                                       |
| :------------------------ | :-------------------------------------------------------------------------------------------------------------------------------- |
| `"auth_user"`             | basic auth username (if provided)                                                                                                 |
| `"client_ip"`             | ip of client                                                                                                                      |
| `"endpoint"`              | path pattern of endpoint                                                                                                          |
| `"handler"`               | one of: `endpoint`, `file`, `spa`                                                                                                 |        
| `"method"`                | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| `"port"`                  | current port accepting request                                                                                                    |
| `"request": {`            | field regarding request information                                                                                               |
| &nbsp;&nbsp;`"bytes"`     | request body size in bytes                                                                                                        |
| &nbsp;&nbsp;`"headers"`   | field regarding keys and values originating from configured keys/header names                                                     |
| &nbsp;&nbsp;`"host"`      | request host                                                                                                                      |
| &nbsp;&nbsp;`"method"`    | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| &nbsp;&nbsp;`"origin"`    | request origin (`<proto>://<host>:<port>`), for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information                                                                                            | 
| &nbsp;&nbsp;`"path"`      | request path                                                                                                                      |
| &nbsp;&nbsp;`"proto"`     | request protocol                                                                                                                  |
| &nbsp;&nbsp;`"status"`    | request status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information  | 
| &nbsp;&nbsp;`"tls" }`     | TLS used `true` or `false`                                                                                                        |
| `"response": {`           | field regarding response information                                                                                              |
| &nbsp;&nbsp;`"bytes"`     | response body size in bytes                                                                                                       |
| &nbsp;&nbsp;`"headers" }` | field regarding keys and values originating from configured keys/header names                                                     |
| `"server"`                | server name (defined in couper file)                                                                                              |
| `"status"`                | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information |
| `"timings": {`            | field regarding timing                                                                                                            |
| &nbsp;&nbsp;`"total" }`   | total time taken                                                                                                                  |
| `“uid"`                   | unique request id configurable in [Settings](./REFERENCE.md#settings-block)                                                       |
| `"url"`                   | complete url (`<proto>://<host>:<port><path>` or `<origin><path>`)                                                                |

### Backend Fields

These are the [Fields](#fields) found in the [Log Type](#log-types) `couper_backend` in addition to the [Common Fields](#common-fields).

| Name                     | Description                                                                                                                       |
| :----------------------- | :-------------------------------------------------------------------------------------------------------------------------------- |
| `"auth_user"`            | backend request basic auth username (if provided)                                                                                 |
| `"backend"`              | configured name, `default` if not provided                                                                                        |
| `"method"`               | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| `"proxy"`                | used system proxy url (if configured), see [Proxy Block](./REFERENCE.md#proxy-block)                                              |
| `"request": {`           | field regarding request information                                                                                               |
| &nbsp;&nbsp;`"bytes"`    | request body size in bytes                                                                                                        |
| &nbsp;&nbsp;`"headers"`  | field regarding keys and values originating from configured keys/header names                                                     |
| &nbsp;&nbsp;`"host"`     | request host                                                                                                                      |
| &nbsp;&nbsp;`"method"`   | http request method, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| &nbsp;&nbsp;`"name"`     | configured request name, `default` if not provided                                                                                | 
| &nbsp;&nbsp;`"origin"`   | request origin, for our purposes excluding `<proto>://` in printing, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Origin) for more information                                                                                                                           | 
| &nbsp;&nbsp;`"path"`     | request path                                                                                                                      |
| &nbsp;&nbsp;`"port"`     | current port accepting request                                                                                                    |
| &nbsp;&nbsp;`"proto" }`  | request protocol                                                                                                                  |
| `"response": {`          | field regarding response information                                                                                              |
| &nbsp;&nbsp;`"headers"`  | field regarding keys and values originating from configured keys/header names                                                     |
| &nbsp;&nbsp;`"status" }` | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information |
| `"status"`               | response status code, see [Mozilla HTTP Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status) for more information |
| `"timings": {`           | field regarding timing                                                                                                            |
| &nbsp;&nbsp;`"dns"`      | time taken by dns                                                                                                                 |
| &nbsp;&nbsp;`"tcp"`      | time taken between attempting and establishing tcp connection                                                                     |
| &nbsp;&nbsp;`"tls"`      | time taken between attempt and success at tls handshake                                                                           |
| &nbsp;&nbsp;`"total"`    | total time taken                                                                                                                  | 
| &nbsp;&nbsp;`"ttfb" }`   | time to first byte/between establishing connection and receiving first byte                                                       |
| `"token_request"`        | entry regarding request for token                                                                                                 |
| `"token_request_retry"`  | how many `token_request` attempts were made                                                                                       |
| `"uid"`                  | unique request id configurable in [Settings](./REFERENCE.md#settings-block)                                                       |
| `"url"`                  | complete url (`<proto>://<host>:<port><path>` or `<origin><path>`)                                                                |
| `"validation"`           | validation result for open api, see [OpenAPI Block](./REFERENCE.md#openapi-block)                                                 |

### Daemon Fields

These are found in the [Log Type](#log-types) `couper_daemon` in addition to the [Common Fields](#common-fields).

| Name                          | Description                                                                                                                |
| :---------------------------- | :------------------------------------------------------------------------------------------------------------------------- |
| `"deadline"`                  | shutdown parameter, see [Health-Check](./REFERENCE.md#health-check)                                                        |
| `"delay"`                     | shutdown parameter, see [Health-Check](./REFERENCE.md#health-check)                                                        |
| `"watch": {`                  | field watching configuration file changes, logs with this field only appear if `watch=true`, more in [Settings](#settings) |
| &nbsp;&nbsp;`"max-retries",`  | maximum retry count, see [Global Options](./CLI.md#global-options)                                                         |
| &nbsp;&nbsp;`"retry-delay" }` | configured delay of each retry, see [Global Options](./CLI.md#global-options)                                              |

## Settings

For more information regarding the usage of these settings, see the documentations regarding [Command Line Interface](./CLI.md#global-options), [Environment Options](./../DOCKER.md#environment-options) and/or [All Settings](./REFERENCE.md#settings-block).

| Feature      | Description                                                                                                                         |
| :----------- | :---------------------------------------------------------------------------------------------------------------------------------- |
| `log-format` | Can be set to either `common` or `json` output format.                                                                              |
| `log-level`  | Can be set to one of: `panic`, `fatal`, `error`, `warn`, `info`, `debug` and `trace`. For more information regarding the usage of each, see the official documentation of [Level](https://pkg.go.dev/github.com/sirupsen/logrus@v1.8.1#Level).                                                                |
| `log-pretty` | Option for `log-format=json` which pretty prints with basic key coloring.                                                           |
| `watch`      | Watch for configuration file changes and reload on modifications. Resulting information manifests in [Daemon Logs](#daemon-fields). |