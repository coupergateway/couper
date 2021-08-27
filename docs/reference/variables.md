# Configuration Reference ~ Variables

The configuration file allows the usage of some predefined variables. There are
two phases when those variables get evaluated:

1. At config load which is currently related to [`couper`](variables/couper.md),
[`env`](variables/env.md) and a simple usage of [Functions](functions.md).
2. During the request/response handling.

## Variables

Click on the variable name to see more details.

| Variable                                              | Description |
| ----------------------------------------------------- | ----------- |
| [`backend_requests`](variables/backend-requests.md)   | Contains a list of all backend requests and their variables. |
| [`backend_responses`](variables/backend-responses.md) | Contains a list of all backend responses and their variables. |
| [`couper`](variables/couper.md)                       | Contains Couper internal variables. |
| [`env`](variables/env.md)                             | Contains information from the environment. |
| [`request`](variables/request.md)                     | Contains original information from the incoming client request and other contextual data. |

## Examples

* [Variables and Expressions](examples.md#variables-and-expressions)

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Parameters](parameters.md)
* &#8674; [Access Control](access-control.md)
