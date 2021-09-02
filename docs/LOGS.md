# Log Functionality Couper

__Logs__ are the information being printed in the console upon the execution of __Couper__. There are a few different [Log Types](#log-types) and several different [Fields](#fields) containing useful information given in said __Logs__ to be remembered, but you can adjust the __Logs'__ appearance and verbosity with a variety of [Features](#features) to make sure that the information you seek is available as easily as possible.

These __Logs__ are still _subject to change_ however. We aspire to make __Couper__ as stable as possible and that includes these __Logs__, so until we believe to have achieved that goal you should expect to see some changes.

## Log Types

| Type                  | Description                                                                                                                                   |
| :-------------------- | :-------------------------------------------------------------------------------------------------------------------------------------------- |
| `couper_access`       | Provides information about the frontend side of things. For information about its [Fields](#fields), see [Access Fields](#access-fields).     |
| `couper_access_tls`   | For information about its [Fields](#fields), see [TLS Access Fields](#tls-access-fields).                                                     |
| `couper_backend`      | Provides information about the backend side of things. For information about its [Fields](#fields), see [Backend Fields](#backend-fields).    |
| `couper_daemon`       | Provides information about the start-up and shut-down of __Couper__ in the form of multiple __Logs__. It is here where the main purpose of the `message` [Field](#fields) lies, as each printed __Log__ of this type will contain a `message` entry regarding either the start-up or shut-down of __Couper__. For information about its [Fields](#fields), see [Daemon Fields](#daemon-fields). |

## Fields

Given the large amount of information contained in several __Logs__, see [Features](#features) to change the format or level of intricacy with which __Logs__ are printed.

### Common Fields

These are a part of all [Log Types](#log-types) and are hence mentioned seperately to avoid unnecessary redundancy.

| Name          | Description                                       |
| :------------ | :------------------------------------------------ |
| `"build”`     | git short hash                                    |
| `"level"`     | configured log level, determines verbosity of log |
| `"message”`   | context based                                     |
| `"timestamp"` | request starting time                             |
| `"type"`      | Type of log                                       |
| `"version"`   | Release version                                   |

### Access Fields

These are found in the [Log Type](#log-types) `couper_access` in addition to the [Common Fields](#common-fields).

| Name                      | Description   |
| :------------------------ | :------------ |
| `"auth_user"`             | basic auth username (if provided) |
| `"client_ip"`             | ip of client |
| `"endpoint"`              | path pattern of endpoint |
| `"handler"`               | type of handler (endpoint, file, spa) |
| `"method"`                | http request method (GET, POST, OPTIONS) (maybe with link to mozilla http request method open source) |
| `"port"`                  | current port accepting request |
| `"request": {`            | field regarding request information |
| &nbsp;&nbsp;`"bytes"`     | body size of request |
| &nbsp;&nbsp;`"headers"`   | field regarding keys and values originating from configured keys/header names |
| &nbsp;&nbsp;`"host"`      | f |
| &nbsp;&nbsp;`"method"`    | mirroring top level |
| &nbsp;&nbsp;`"origin"`    | . | 
| &nbsp;&nbsp;`"path"`      | . |
| &nbsp;&nbsp;`"proto"`     | . |
| &nbsp;&nbsp;`"status"`    | . | 
| &nbsp;&nbsp;`"tls" }`     | . |
| `"response": {`           | field regarding response info |
| &nbsp;&nbsp;`"bytes"`     | body size of response |
| &nbsp;&nbsp;`"headers" }` | field regarding keys and values originating from configured keys/header names |
| `"server"`                | Server name (defined in couper file) |
| `"status"`                | response status code |
| `"timings": {`            | field regarding timing |
| &nbsp;&nbsp;`"total" }`   | total time taken |
| `“uid"`                   | unique request id configurable in unique id settings (maybe link) |
| `"url"`                   | url |

### TLS Access Fields

These are found in the [Log Type](#log-types) `couper_access_tls` in addition to the [Common Fields](#common-fields).

| Name                      | Description   |
| :------------------------ | :------------ |
|                           | . |

### Backend Fields

These are the [Fields](#fields) found in the [Log Type](#log-types) `couper_backend` in addition to the [Common Fields](#common-fields).

| Name                      | Description   |
| :------------------------ | :------------ |
| `"auth_user"`             | backend request basic auth username (if provided) |
| `"backend"`               | name can be set, “default” otherwise |
| `"method"`                | http request method (GET, POST, OPTIONS) (maybe with link to mozilla http request method open source) |
| `"proxy"`                 | . |
| `"request": {`            | . |
| &nbsp;&nbsp;`"bytes"`     | . | 
| &nbsp;&nbsp;`"headers"`   | . |
| &nbsp;&nbsp;`"host"`      | . |
| &nbsp;&nbsp;`"method"`    | . | 
| &nbsp;&nbsp;`"name"`      | . | 
| &nbsp;&nbsp;`"origin"`    | . | 
| &nbsp;&nbsp;`"path"`      | . | 
| &nbsp;&nbsp;`"port"`      | . |
| &nbsp;&nbsp;`"proto" }`   | . |
| `"response": {`           | . |
| &nbsp;&nbsp;`"headers"`   | . |
| &nbsp;&nbsp;`"status" }`  | . |
| `"status"`                | . | 
| `"timings": {`            | . |
| &nbsp;&nbsp;`"dns"`       | connect to which ip |
| &nbsp;&nbsp;`"tcp"`       | connect time |
| &nbsp;&nbsp;`"tls"`       | time after connection for handshake |
| &nbsp;&nbsp;`"total"`     | . | 
| &nbsp;&nbsp;`"ttfb" }`    | time to first byte |
| `"token_request"`         | link to documentation |
| `"token_request_retry"`   |  How many attempts at request, documentation |
| `"uid"`                   | . |
| `"url"`                   | . |
| `"validation"`            | validation result for open api (link to config) |

### Daemon Fields

These are found in the [Log Type](#log-types) `couper_daemon` in addition to the [Common Fields](#common-fields).

| Name                      | Description   |
| :------------------------ | :------------ |
|                           |               |

## Features

| Feature   | Description   |
| :-------- | :------------ |
|           |               |