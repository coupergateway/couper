- [Introduction](#introduction)
  - [Log Types](#log-types)
  - [Fields](#fields)
    - [Access Fields](#access-fields)
    - [TLS Access Fields](#tls-access-fields)
    - [Backend Fields](#backend-fields)
    - [Daemon Fields](#daemon-fields)
  - [Features](#features)

# Introduction

_Logs_ are the information being printed in the console upon the execution of Couper. There are a few different [Log Types](#log-types) and several different [Fields](#fields) containing useful information given in said _Logs_ to be remembered, but you can adjust the _Logs'_ appearance and verbosity with a variety of [Features](#features) to make sure that the information you seek is available as easily as possible.

These _Logs_ are however still subject to change. We aspire to make Couper as stable as possible and that includes the _Logs_, so until we believe to have achieved that goal you should expect to see some changes.

## Log Types

| Type                | Description                                                                                                                                     |
| :------------------ | :---------------------------------------------------------------------------------------------------------------------------------------------- |
| `couper_access`     | Provides information about the frontend side of things. For information about its [Fields](#fields), see [Access Fields](#access-fields).       |
| `couper_access_tls` | Provides information about connection to TLS proxies. For information about its [Fields](#fields), see [TLS Access Fields](#tls-access-fields). |
| `couper_backend`    | Provides information about the backend side of things. For information about its [Fields](#fields), see [Backend Fields](#backend-fields).      |
| `couper_daemon`     | Provides information about the start-up and shut-down of Couper in the form of multiple _Logs_. It is here where the main purpose of the `message` [Field](#fields) lies, as each printed _Log_ of this type will contain a `message` entry regarding either the start-up or shut-down of Couper. For information about its [Fields](#fields), see [Daemon Fields](#daemon-fields).                                                                                                                         |

## Fields

Given the large amount of information contained in several _Logs_, see [Features](#features) to change the format or level of intricacy with which _Logs_ are printed.

### Common Fields

These are a part of all [Log Types](#log-types) and are hence mentioned seperately to avoid unnecessary redundancy.

| Name          | Description                                       |
| :------------ | :------------------------------------------------ |
| `"build”`     | git short hash                                    |
| `"level"`     | configured log level, determines verbosity of log |
| `"message”`   | context based, mainly used in `couper_daemon`     |
| `"timestamp"` | request starting time                             |
| `"type"`      | [Log Type](#log-types)                            |
| `"version"`   | release version                                   |

### Access Fields

These are found in the [Log Type](#log-types) `couper_access` in addition to the [Common Fields](#common-fields).

| Name                      | Description                                                                                                                  |
| :------------------------ | :--------------------------------------------------------------------------------------------------------------------------- |
| `"auth_user"`             | basic auth username (if provided)                                                                                            |
| `"client_ip"`             | ip of client                                                                                                                 |
| `"endpoint"`              | path pattern of endpoint                                                                                                     |
| `"handler"`               | one of: `endpoint`, `file`, `spa`                                                                                            |
| `"method"`                | http request method, see [Mozilla Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| `"port"`                  | current port accepting request                                                                                               |
| `"request": {`            | field regarding request information                                                                                          |
| &nbsp;&nbsp;`"bytes"`     | body size of request                                                                                                         |
| &nbsp;&nbsp;`"headers"`   | field regarding keys and values originating from configured keys/header names                                                |
| &nbsp;&nbsp;`"host"`      | host of request                                                                                                              |
| &nbsp;&nbsp;`"method"`    | http request method, see [Mozilla Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| &nbsp;&nbsp;`"origin"`    | origin of request                                                                                                            | 
| &nbsp;&nbsp;`"path"`      | path of request                                                                                                              |
| &nbsp;&nbsp;`"proto"`     | protocol of request                                                                                                          |
| &nbsp;&nbsp;`"status"`    | status of request                                                                                                            | 
| &nbsp;&nbsp;`"tls" }`     | TLS used `true`or `false`                                                                                                    |
| `"response": {`           | field regarding response info                                                                                                |
| &nbsp;&nbsp;`"bytes"`     | body size of response                                                                                                        |
| &nbsp;&nbsp;`"headers" }` | field regarding keys and values originating from configured keys/header names                                                |
| `"server"`                | server name (defined in couper file)                                                                                         |
| `"status"`                | response status code                                                                                                         |
| `"timings": {`            | field regarding timing                                                                                                       |
| &nbsp;&nbsp;`"total" }`   | total time taken                                                                                                             |
| `“uid"`                   | unique request id configurable in [Settings](../docs/REFERENCE.md#settings-block)                                            |
| `"url"`                   | complete url (`<proto>://<host>:<port><path>` or `<proto>://<origin><path>`)                                                 |

### TLS Access Fields

These are found in the [Log Type](#log-types) `couper_access_tls` in addition to the [Common Fields](#common-fields).

| Name                      | Description   |
| :------------------------ | :------------ |
| .                         | . |

### Backend Fields

These are the [Fields](#fields) found in the [Log Type](#log-types) `couper_backend` in addition to the [Common Fields](#common-fields).

| Name                     | Description                                                                                                                  |
| :----------------------- | :--------------------------------------------------------------------------------------------------------------------------- |
| `"auth_user"`            | backend request basic auth username (if provided)                                                                            |
| `"backend"`              | name can be set, “default” otherwise                                                                                         |
| `"method"`               | http request method, see [Mozilla Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| `"proxy"`                | . |
| `"request": {`           | field regarding request information                                                                                          |
| &nbsp;&nbsp;`"bytes"`    | body size of request                                                                                                         |
| &nbsp;&nbsp;`"headers"`  | field regarding keys and values originating from configured keys/header names                                                |
| &nbsp;&nbsp;`"host"`     | host of request                                                                                                              |
| &nbsp;&nbsp;`"method"`   | http request method, see [Mozilla Reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods) for more information |
| &nbsp;&nbsp;`"name"`     | . | 
| &nbsp;&nbsp;`"origin"`   | origin of request                                                                                                            | 
| &nbsp;&nbsp;`"path"`     | path of request                                                                                                              |
| &nbsp;&nbsp;`"port"`     | current port accepting request                                                                                               |
| &nbsp;&nbsp;`"proto"`    | protocol of request                                                                                                          |
| `"response": {`          | field regarding response information                                                                                         |
| &nbsp;&nbsp;`"headers"`  | field regarding keys and values originating from configured keys/header names                                                |
| &nbsp;&nbsp;`"status" }` | status of response                                                                                                           |
| `"status"`               | status of response                                                                                                           | 
| `"timings": {`           | field regarding timing                                                                                                       |
| &nbsp;&nbsp;`"dns"`      | time taken by dns                                                                                                            |
| &nbsp;&nbsp;`"tcp"`      | time taken between attempting and establishing connection                                                                    |
| &nbsp;&nbsp;`"tls"`      | time taken between attempt and success at tls handshake                                                                      |
| &nbsp;&nbsp;`"total"`    | total time taken                                                                                                             | 
| &nbsp;&nbsp;`"ttfb" }`   | time to first byte/between establishing connection and receiving first byte                                                  |
| `"token_request"`        | . |
| `"token_request_retry"`  | how many attempts at `token_request`                                                                                         |
| `"uid"`                  | . |
| `"url"`                  | . |
| `"validation"`           | validation result for open api (link to config pending) |

### Daemon Fields

These are found in the [Log Type](#log-types) `couper_daemon` in addition to the [Common Fields](#common-fields).

| Name                          | Description                                                                                                                |
| :---------------------------- | :------------------------------------------------------------------------------------------------------------------------- |
| `"deadline"`                  | . |
| `"delay"`                     | . |
| `"watch": {`                  | field watching configuration file changes, logs with this field only appear if `watch=true`, more in [Features](#features) |
| &nbsp;&nbsp;`"max-retries",`  | maximum retry count, see [Global Options](../docs/CLI.md#global-options)                                                   |
| &nbsp;&nbsp;`"retry-delay" }` | configured delay of each retry, see [Global Options](../docs/CLI.md#global-options)                                        |

## Features

For more information regarding the usage of these features, see the documentations regarding the [Command Line Interface](../docs/CLI.md#global-options), [Enviromment Options](../DOCKER.md#environment-options) and/or [Settings](REFERENCE.md#settings-block).

| Feature      | Description                                                                                                                         |
| :----------- | :---------------------------------------------------------------------------------------------------------------------------------- |
| `log-format` | Can be set to either `common` or `json` output format.                                                                              |
| `log-level`  | Can be set to one of: `panic`, `fatal`, `error`, `warn`, `info`, `debug` and `trace`. For more information regarding the usage of each, see the official documentation of [Level](https://pkg.go.dev/github.com/sirupsen/logrus@v1.8.1#Level).                                                                |
| `log-pretty` | Option for `log-format=json` which pretty prints with basic key coloring.                                                           |
| `watch`      | Watch for configuration file changes and reload on modifications. Resulting information manifests in [Daemon Logs](#daemon-fields). |