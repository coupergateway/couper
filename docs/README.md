# Couper Docs - Version 0.5

## Table of contents

* [Introduction](#introduction)
* [Core concepts](#core-concepts)
* [Configuration file](#configuration-file)
  * [Syntax](#syntax)
  * [File name](#file-name)
  * [Basic file structure](#basic-file-structure)
  * [Variables](#variables)
    * [`env`](#env-variable)
    * [`req`](#req-client-request-variable)
    * [`bereq`](#bereq-modified-backend-request-variable)
    * [`beresp`](#beresp-original-backend-response-variable)
    * [Variable example](#variable-example)
  * [Expressions](#expressions)
  * [Functions](#functions)
* [Reference](#reference)
  * [The `server` block](#the-server-block)
    * [The `files` block](#the-files-block)
    * [The `spa` block](#the-spa-block)
    * [The `api` block](#the-api-block)
    * [The `endpoint` block](#the-endpoint-block)
      * [Query parameter](#query-parameter)
      * [Request header](#request-header)
      * [Response header](#response-header)
      * [Path parameter](#path-parameter)
    * [The `backend` block](#the-backend-block)
      * [The `openapi` block](#the-openapi-block)
    * [The `cors` block](#the-cors-block)
    * [The `access_control`](#the-access_control)
  * [The `definitions` block](#the-definitions-block)
    * [The `basic_auth` block](#the-basic_auth-block)
    * [The `jwt` block](#the-jwt-block)
  * [The `settings` block](#the-settings-block)
  * [Health-Check](#health-check)
* [Examples](#examples)
  * [Request routing](#request-routing-example)
  * [Routing configuration](#routing-configuration-example)
  * [Web serving configuration](#web-serving-configuration-example)
  * [`access_control` configuration](#access_control-configuration-example)
  * [`hosts` configuration](#hosts-configuration-example)
  * [Referencing and overwriting](#referencing-and-overwriting-example)

## Introduction

Couper is a frontend gateway especially designed to support building and running
API-driven web projects. Acting as a proxy component it connects clients with (micro)
services and adds access control and observability to the project. Couper does not
need any special development skills and offers easy configuration and integration.

## Core concepts

![overview](./overview.png)

| Concept / Feature  | Description |
|:-------------------|:------------|
| Client(s)          | Browser, App or API Client that sends requests to Couper. |
| Web Serving        | Couper supports file serving and web serving for SPA assets. |
| API                | Configuration block that bundles endpoints under a certain base path. |
| Access Control     | Couper handles access control for incoming client requests. |
| Endpoint           | Configuration block that specifies how (and if) requests are sent to backend service(s) after they reach Couper. |
| Backend            | Configuration block that specifies the connection to a local/remote backend service. |
| Logging            | Couper provides standard logs for analysis and monitoring. |
| Backend Service(s) | External API or micro services where Couper fetches data from. |

## Configuration file

### Syntax

The syntax for Couper's configuration file is
[HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax),
a configuration language by HashiCorp.

### File name

The file-ending of your configuration file should be .hcl to have syntax highlighting
within your IDE.

The file name defaults to `couper.hcl` in your working directory. This can be
changed with the `-f` command-line flag. With `-f /opt/couper/my_conf.hcl` couper
changes the working directory to `/opt/couper` and loads `my_conf.hcl`.

### Basic file structure

Couper's configuration file consists of nested configuration blocks that configure
web serving and routing of the gateway. Access control is controlled by an
`access_control` attribute that can be set for blocks.

For orientation compare the following example and the information below:

```hcl
server "my_project" {
  files { ... }

  spa { ... }

  api {
    access_control = ["foo"]
    endpoint "/bar" {
      backend { ... }
    }
  }
}

definitions { ... }
```

* `server` main configuration block(s)
  * `files` configuration block for file serving
  * `spa` configuration block for web serving (spa assets)
  * `api` configuration block(s) that bundles endpoints under a certain base path or `access_control` list
  * `access_control` attribute that sets access control for a block context
  * `endpoint` configuration block for Couper's entry points
  * `backend` configuration block for connection to local/remote backend service(s)
* `definitions` block for predefined configurations, that can be referenced
* `settings` block for server configuration which applies to the running instance

### Variables

The configuration file allows the use of some predefined variables. There are
two phases when those variables get evaluated. The first phase is at config load
which is currently related to `env` and **function** usage. The second evaluation
will happen during the request/response handling.

* `env` are the environment variables
* `req` is the client request
* `bereq` is the modified backend request
* `beresp` is the original backend response

Most fields are self-explanatory (compare tables below).

#### `env` variable

Environment variables can be accessed everywhere within the configuration file
since these references get evaluated at start.

#### `req` (client request) variable

| Variable                  | Description |
|:--------------------------|:------------|
| `id`                      | unique request id |
| `method`                  | HTTP method |
| `path`                    | URL path |
| `endpoint`                | matched endpoint pattern |
| `headers.<name>`          | HTTP request header value for requested lower-case key |
| `cookies.<name>`          | value from `Cookie` request header for requested key (&#9888; last wins!) |
| `query.<name>`            | query parameter values (&#9888; last wins!) |
| `path_params.<name>`      | value from a named path parameter defined within an endpoint path label |
| `post.<name>`             | post form parameter |
| `json_body.<name>`        | Access json decoded object properties. Media type must be `application/json`. |
| `ctx.<name>.<claim_name>` | request context containing claims from JWT used for [access control](#the-access_control), `<name>` being the [`jwt` block's](#the-jwt-block) label and `claim_name` being the claim's name |

#### `bereq` (modified backend request) variable

| Variable                  | Description |
|:--------------------------|:------------|
| `id`                      | unique request id |
| `method`                  | HTTP method |
| `path`                    | URL path |
| `headers.<name>`          | HTTP request header value for requested lower-case key |
| `cookies.<name>`          | value from `Cookie` request header for requested key (&#9888; last wins!) |
| `query.<name>`            | query parameter values (&#9888; last wins!) |
| `post.<name>`             | post form parameter |
| `ctx.<name>.<claim_name>` | request context containing claims from JWT used for [access control](#the-access_control), `<name>` being the [`jwt` block's](#the-jwt-block) label and `claim_name` being the claim's name |
| `url`                     | backend origin URL |

#### `beresp` (original backend response) variable

| Variable           | Description |
|:-------------------|:------------|
| `status`           | HTTP status code |
| `headers.<name>`   | HTTP response header value for requested lower-case key |
| `cookies.<name>`   | Value from `Set-Cookie` response header for requested key (&#9888; last wins!) |
| `json_body.<name>` | Access json decoded object properties. Media type must be `application/json`. |

##### Variable Example

An example to send an additional header with client request header to a configured
backend and gets evaluated on per request basis:

```hcl
server "variables-srv" {
  api {
    endpoint "/" {
      backend "my_backend_definition" {
        set_request_headers = {
          x-env-user = env.USER
          user-agent = "myproxyClient/${req.headers.app-version}"
          x-uuid = req.id
        }
      }
    }
  }
}
```

### Expressions

Since we use HCL2 for our configuration, we are able to use attribute values as
expression:

```hcl
// Arithmetic with literals and application-provided variables
sum = 1 + addend

// String interpolation and templates
message = "Hello, ${name}!"

// Application-provided functions
shouty_message = upper(message)
```

### Functions

Functions are little helper methods which are registered for every hcl evaluation
context.

| Name               | Description |
|:-------------------|:------------|
| `base64_decode`    | base64_decode decodes Base64 data, as specified in RFC 4648         |
| `base64_encode`    | base64_encode encodes Base64 data, as specified in RFC 4648         |
| `coalesce`         | coalesce returns the first of the given arguments that is not null. |
| `json_decode`      | json_decode parses the given JSON string and, if it is valid, returns the value it represents. |
| `json_encode`      | json_encode returns a JSON serialization of the given value.        |
| `to_lower`         | to_lower is a Function that converts a given string to lowercase.   |
| `to_upper`         | to_upper is a Function that converts a given string to uppercase.   |

Example usage:

```hcl
my_attribute = base64_decode("aGVsbG8gd29ybGQK")

my_json = json_encode({
  value-a: beresp.json_body.origin
  value-b: ["item1", "item2"]
})
```

## Reference

### The `server` block

The `server` block is the main configuration block of Couper's configuration file.
It has an optional label and a `hosts` attribute. Nested blocks are `files`, `spa`
and `api`. You can declare `access_control` for the `server` block.
`access_control` is inherited by nested blocks.

| Name                                        | Description |
|:--------------------------------------------|:------------|
| context                                     | none |
| *label*                                     | optional |
| `hosts`                                     | <ul><li>list  </li><li>&#9888; mandatory, if there is more than one `server` block</li><li>*example:* `hosts = ["example.com", "..."]`</li><li>you can add a specific port to your host <br> *example:* `hosts = ["localhost:9090"]` </li><li>default port is `8080`</li><li>only **one** `hosts` attribute per `server` block is allowed</li><li>compare the hosts [example](#hosts-configuration-example) for details</li></ul> |
| `error_file`                                | <ul><li>location of the error template file</li><li>*example:* `error_file = "./my_error_page.html"`</li></ul> |
| [**`access_control`**](#the-access_control) | <ul><li>sets predefined `access_control` for `server` block</li><li>*example:* `access_control = ["foo"]`</li><li>&#9888; inherited</li></ul> |
| [**`files`**](#the-files-block) block       | configures file serving |
| [**`spa`**](#the-spa-block) block           | configures web serving for spa assets |
| [**`api`**](#the-api-block) block(s)        | configures routing and backend connection(s) |

### The `files` block

The `files` block configures your document root, and the location of your error
document.

| Name                                        | Description |
|:--------------------------------------------|:------------|
| context                                     | `server` block |
| `document_root`                             | <ul><li>location of the document root</li><li>*example:* `document_root = "./htdocs"`</li></ul> |
| `error_file`                                | <ul><li>location of the error template file</li><li>*example:* `error_file = "./my_error_page.html"`</li></ul> |
| [**`access_control`**](#the-access_control) | <ul><li>sets predefined `access_control` for `files` block context</li><li>*example:* `access_control = ["foo"]`</li></ul> |

### The `spa` block

The `spa` block configures the location of your bootstrap file and your SPA paths.

| Name                                        | Description |
|:--------------------------------------------|:------------|
| context                                     | `server` block |
| `bootstrap_file`                            | <ul><li>location of the bootstrap file</li><li>*example:* `bootstrap_file = "./htdocs/index.html"`</li></ul>|
| `paths`                                     | <ul><li>list of SPA paths that need the bootstrap file</li><li>*example:* `paths = ["/app/**"]`</li></ul> |
| [**`access_control`**](#the-access_control) | <ul><li>sets predefined `access_control` for `spa` block context</li><li>*example:* `access_control = ["foo"]`</li></ul> |

### The `api` block

The `api` block contains all information about endpoints, and the connection to
remote/local backend service(s) (configured in the nested `endpoint` and `backend`
blocks). You can add more than one `api` block to a `server` block. If an error
occurred for api endpoints the response gets processed as json error with an error
body payload. This can be customized via `error_file`.

| Name                                        | Description |
|:--------------------------------------------|:------------|
| context                                     | `server` block |
| `base_path`                                 | <ul><li>optional</li><li>*example:* `base_path = "/api"`</li></ul> |
| `error_file`                                | <ul><li>location of the error template file</li><li>*example:* `error_file = "./my_error_body.json"`</li></ul> |
| [**`access_control`**](#the-access_control) | <ul><li>sets predefined `access_control` for `api` block context</li><li>&#9888; inherited by all endpoints in `api` block context</li></ul> |
| [**`backend`**](#the-backend-block) block   | <ul><li>configures connection to a local/remote backend service for `api` block context</li><li>&#9888; only one `backend` block per `api` block<li>&#9888; inherited by all endpoints in `api` block context</li></ul> |
| [**`endpoint`**](#the-endpoint-block) block | configures specific endpoint for `api` block context |
| [**`cors`**](#the-cors-block) block         | configures CORS behavior for `api` block context |

### The `cors` block

The CORS block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

| Name                       | Description |
|:---------------------------|:------------|
| context                    | `api`block |
| `allowed_origins`          | <ul><li>(list of) allowed origin(s)</li><li> can be either a string with a single specific origin (e.g. `https://www.example.com`)</li><li> or `*` (all origins allowed) </li><li>or an array of specific origins (`["https://www.example.com", "https://www.another.host.org"]`)</li></ul> |
| `allow_credentials = true` | if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` headers) |
| `max_age`                  |  <ul><li>indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response headers</li><li> can be cached (string with time unit, e.g. `"1h"`) </li></ul> |

### The `endpoint` block

Endpoints define the entry points of Couper. The mandatory *label* defines the path
suffix for the incoming client request. The `path` attribute changes the path for
the outgoing request (compare [request routing example](#request-routing-example)).
Each `endpoint` must have at least one `backend` which can be declared in the `api`
context above or inside an `endpoint`.

| Name                                          | Description |
|:----------------------------------------------|:------------|
| context                                       | `server` and `api` block |
| *label*                                       | <ul><li>&#9888; mandatory</li><li>defines the path suffix for incoming client requests</li><li>*example:* `endpoint "/dashboard" {`</li><li>incoming client request: `example.com/api/dashboard`</li></ul> |
| `path`                                        | <ul><li>changeable part of upstream URL</li><li>changes the path suffix of the outgoing request</li></ul> |
| [**`access_control`**](#the-access_control)   | sets predefined `access_control` for `endpoint` |
| [**`backend`**](#the-backend-block) block     | configures connection to a local/remote backend service for `endpoint` |
| [**`remove_query_params`**](#query-parameter) | <ul><li>a list of query parameters to be removed from the upstream request URL</li></ul> |
| [**`set_query_params`**](#query-parameter)    | <ul><li>key/value(s) pairs to set query parameters in the upstream request URL</li></ul> |
| [**`add_query_params`**](#query-parameter)    | <ul><li>key/value(s) pairs to add query parameters to the upstream request URL</li></ul> |

#### Query parameter

Couper offers three attributes to manipulate the query parameter. The query
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

* `remove_query_params` a list of query parameters to be removed from the upstream
  request URL.
* `set_query_params` key/value(s) pairs to set query parameters in the upstream
  request URL.
* `add_query_params` key/value(s) pairs to add query parameters to the upstream
  request URL.

All `*_query_params` are collected and executed from: `definitions.backend`, `endpoint`,
`endpoint.backend` (if refined).

```hcl
server "my_project" {
  api {
    endpoint "/" {
      backend = "example"
    }
  }
}

definitions {
  backend "example" {
    origin = "http://example.com"

    remove_query_params = ["a", "b"]

    set_query_params = {
      string = "string"
      multi = ["foo", "bar"]
      "${req.headers.example}" = "yes"
    }

    add_query_params = {
      noop = req.headers.noop
      null = null
      empty = ""
    }
  }
}
```

#### Request header

Couper offers three attributes to manipulate the request header fields. The header
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

* `remove_request_headers` a list of request header to be removed from the upstream
  request.
* `set_request_headers` key/value(s) pairs to set request header in the upstream
  request.
* `add_request_headers` key/value(s) pairs to add request header to the upstream
  request.

All `*_request_headers` are collected and executed from: `definitions.backend`,
`endpoint.backend` (if refined).

#### Response header

Couper offers three attributes to manipulate the response header fields. The header
attributes can be defined unordered within the configuration file but will be
executed ordered as follows:

* `remove_response_headers` a list of response header to be removed from the client
  response.
* `set_response_headers` key/value(s) pairs to set response header in the client
  response.
* `add_response_headers` key/value(s) pairs to add response header to the client
  response.

All `*_response_headers` are collected and executed from: `definitions.backend`,
`endpoint.backend` (if refined).

#### Path parameter

An endpoint label could be defined as `endpoint "/app/{section}/{project}/view" { ... }`
to access the named path parameter `section` and `project` via `req.path_param.*`.
The values would map as following for the request path: `/app/nature/plant-a-tree/view`:

| Variable                  | Value          |
|:--------------------------|:---------------|
| `req.path_params.section` | `nature`       |
| `req.path_params.project` | `plant-a-tree` |

### The `backend` block

A `backend` defines the connection to a local/remote backend service. Backends
can be defined globally in the `api` block for all endpoints of an API or inside
an `endpoint`. An `endpoint` must have (at least) one `backend`. You can also
define backends in the `definitions` block and use the mandatory *label* as reference.

| Name                                              | Description | Default |
|:--------------------------------------------------|:------------|:--------|
| context                                           | <ul><li>`server` block</li><li>`api` block</li><li>`endpoint` block</li><li>`definitions` block (reference purpose)</li></ul> ||
| *label*                                           | <ul><li>&#9888; mandatory, when declared in `server` block</li><li>&#9888; mandatory, when declared in `api` block</li><li>&#9888; mandatory, when declared in `definitions` block</li></ul> ||
| `base_path`                                       | <ul><li>`base_path` for backend</li><li>won\`t change for `endpoint`</li></ul> ||
| `basic_auth`                                      | basic_auth for the upstream request in format `username:password` ||
| `disable_certificate_validation`                  | Disables the peer certificate validation. | `false` |
| `disable_connection_reuse`                        | Disables reusage of connections | `false` |
| `hostname`                                        | value of the HTTP host header field for the `origin` request. Since `hostname` replaces the request host the value will also be used for a server identity check during a TLS handshake with the origin. ||
| `http2`                                           | `boolean`, enables the HTTP2 support | `false` |
| `max_connections`                                 | Describes the maximum number of concurrent connections in any state (*active* or *idle*) to the `origin`. | `0` (no limit) |
| `origin`                                          | URL to connect to for backend requests </br> &#9888; must start with the scheme `http://...` ||
| `path`                                            | changeable part of upstream URL ||
| `proxy`                                           | A Proxy URL for the related origin request. Example: `http://SERVER-IP_OR_NAME:PORT` | `none` |
| `request_body_limit`                              | Limit to configure the maximum buffer size while accessing `req.post` or `req.json_body` content. Valid units are: `KiB, MiB, GiB`. | `64MiB` |
| [**`add_request_headers`**](#request-header)      | header map to define additional header values for the `origin` request ||
| [**`add_response_headers`**](#response-header)    | same as `add_request_headers` for the client response ||
| [**`remove_request_headers`**](#request-header)   | header list to define header to be removed from the `origin` request ||
| [**`remove_response_headers`**](#response-header) | same as `remove_request_headers` for the client response ||
| [**`set_request_headers`**](#request-header)      | header map to override header for the `origin` request ||
| [**`set_response_headers`**](#response-header)    | same as `set_request_headers` for the client response ||
| [**`openapi`**](#the-openapi-block)               | Definition for validating outgoing requests to the `origin` and incoming responses from the `origin`. ||
| [**`remove_query_params`**](#query-parameter)     | a list of query parameters to be removed from the upstream request URL ||
| [**`set_query_params`**](#query-parameter)        | key/value(s) pairs to set query parameters in the upstream request URL ||
| [**`add_query_params`**](#query-parameter)        | key/value(s) pairs to add query parameters to the upstream request URL ||
| **Timings**                                       | **valid time units are: "ns", "us" (or "µs"), "ms", "s", "m", "h"**  | **Default** |
| `connect_timeout`                                 | The total timeout for dialing and connect to the origins network address. | `10s` |
| `timeout`                                         | the total deadline duration a backend request has for write and read/pipe | `300s` |
| `ttfb_timeout`                                    | Time to first byte timeout describes the duration from writing the full request to the `origin` to receiving the answer. | `60s` |

### The `access_control`

The configuration of access control is twofold in Couper: You define the particular
type (such as `jwt` or `basic_auth`) in `definitions`, each with a distinct label.
Anywhere in the `server` block those labels can be used in the `access_control`
list to protect that block. &#9888; access rights are inherited by nested blocks.
You can also disable `access_control` for blocks. By typing `disable_access_control = ["bar"]`,
the `access_control` type `bar` will be disabled for the corresponding block context.

Compare the `access_control` [example](#access_control-configuration-example) for details.

#### The `openapi` block

The `openapi` block configures the backends proxy behaviour to validate outgoing
and incoming requests to and from the origin. Preventing the origin from invalid
requests, and the Couper client from invalid answers. An example can be found
[here](https://github.com/avenga/couper-examples/blob/master/backend-validation/README.md).
To do so Couper uses the [OpenAPI 3 standard](https://www.openapis.org/) to load
the definitions from a given document defined with the `file` attribute.

| Name                         | Description                                        | Default   |
|:-----------------------------|:---------------------------------------------------|:----------|
| context                      | `backend` block                                    |           |
| `file`                       | OpenAPI yaml definition file                       | mandatory |
| `ignore_request_violations`  | log request validation results, skip err handling  | `false`   |
| `ignore_response_violations` | log response validation results, skip err handling | `false`   |

**Caveats**: While ignoring request violations an invalid method or path would
lead to a non-matching *route* which is still required for response validations.
In this case the response validation will fail if not ignored too.

### The `definitions` block

Use the `definitions` block to define configurations you want to reuse.
`access_control` is **always** defined in the `definitions` block.

#### The `basic_auth` block

The `basic_auth` block let you configure basic auth for your gateway. Like all
`access_control` types, the `basic_auth` block is defined in the `definitions`
block and can be referenced in all configuration blocks by its mandatory *label*.

If both `user`/`password` and `htpasswd_file` are configured, the incoming
credentials from the `Authorization` request header are checked against `user`/`password`
if the user matches, and against the data in the file referenced by `htpasswd_file`
otherwise.

| Name            | Description |
|:----------------|:------------|
| context         | `definitions` block |
| *label*         | <ul><li>&#9888; mandatory</li><li>always defined in `definitions` block</li></ul> |
| `user`          | The user name |
| `password`      | The corresponding password |
| `htpasswd_file` | The htpasswd file |
| `realm`         | The realm to be sent in a `WWW-Authenticate` response header |

#### The `jwt` block

The `jwt` block let you configure JSON Web Token access control for your gateway.
Like all `access_control` types, the `jwt` block is defined in the `definitions`
block and can be referenced in all configuration blocks by its mandatory *label*.

| Name                      | Description |
|:--------------------------|:------------|
| context                   | `definitions` block |
| *label*                   | <ul><li>&#9888; mandatory</li><li>always defined in `definitions` block</li></ul> |
| `cookie = "AccessToken"`  | read `AccessToken` key to gain the token value from a cookie |
| `header = "Authorization` | &#9888; implies Bearer if `Authorization` is used, otherwise any other header name can be used |
| `header = "API-Token`     | alternative header source for our token |
| `key`                     | public key for `RS*` variants or the secret for `HS*` algorithm |
| `key_file`                | optional file reference instead of `key` usage |
| `signature_algorithm`     | valid values are: `RS256` `RS384` `RS512` `HS256` `HS384` `HS512` |
| **`claims`**              | equals/in comparison with JWT payload |

### The `settings` block

The `settings` block let you configure the more basic and global behavior of your
gateway instance.

| Name                | Description | Default |
|:--------------------|:------------|:--------|
| `health_path`       | health path which is available for all configured server and ports | `/healthz` |
| `no_proxy_from_env` | Disables the connect hop to configured [proxy via environment](https://godoc.org/golang.org/x/net/http/httpproxy). | `false` |
| `default_port`      | port which will be used if not explicitly specified per host within the [`hosts`](#the-server-block) list | `8080` |
| `log_format`        | switch for tab/field based colored view or json log lines | `common` |
| `xfh`               | option to use the `X-Forwarded-Host` header as the request host | `false` |
| `request_id_format` | if set to `uuid4` a rfc4122 uuid is used for `req.id` and related log fields | `common` |

### Health-Check

The health check will answer a status `200 OK` on every port with the configured
`health_path`. As soon as the gateway instance will receive a `SIGINT` or `SIGTERM`
the check will return a status `500 StatusInternalServerError`. A shutdown delay
of `5s` allows the server to finish all running requests and gives a load-balancer
time to pick another gateway instance. After this delay the server goes into
shutdown mode with a deadline of `5s` and no new requests will be accepted. The
shutdown timings cannot be configured at this moment.

## Examples

See the official Couper's examples and tutorials
[repository](https://github.com/avenga/couper-examples), too.

### Request routing example

![routing_example](./routing_example.png)

| No. | Configuration source                              |
|:----|:--------------------------------------------------|
| 1   | `hosts` attribute in `server` block               |
| 2   | `base_path` attribute in `api` block              |
| 3   | *label* of `endpoint` block                       |
| 4   | `origin` attribute in `backend` block             |
| 5   | `base_path` attribute in `backend`                |
| 6   | `path` attribute in `endpoint` or `backend` block |

### Routing configuration example

```hcl
api "my_api" {
  base_path = "/api/novoconnect"

  endpoint "/login/**" {
    # incoming request: .../login/foo
    # implicit proxy
    # outgoing request: http://identityprovider:8080/login/foo
    backend {
      origin = "http://identityprovider:8080"
    }
  }

  endpoint "/cart/**" {
      # incoming request: .../cart/items
      # outgoing request: http://cartservice:8080/api/v1/items
      path = "/api/v1/**"
      backend {
        origin = "http://cartservice:8080"
      }

      endpoint "/account/{id}" {
        # incoming request: .../account/brenda
        # outgoing request: http://accountservice:8080/user/brenda/info
        backend {
          path = "/user/${req.param.id}/info"
          origin = "http://accountservice:8080"
        }
      }
    }
  }
```

### Web serving configuration example

```hcl
server "my_project" {
  files {
    document_root = "./htdocs"
    error_file = "./my_custom_error_page.html"
  }

  spa {
    bootstrap_file = "./htdocs/index.html"
    paths = [
      "/app/**",
      "/profile/**"
    ]
  }
}
```

### `access_control` configuration example

```hcl
server {
  access_control = ["ac1"]
  files {
    access_control = ["ac2"]
  }

  spa {
    bootstrap_file = "myapp.html"
  }

  api {
    access_control = ["ac3"]
    endpoint "/foo" {
      disable_access_control = ["ac3"]
    }
    endpoint "/bar" {
      access_control = ["ac4"]
    }
  }
}

definitions {
  basic_auth "ac1" { ... }
  jwt "ac2" { ... }
  jwt "ac3" { ... }
  jwt "ac4" { ... }
}
```

The following table shows which `access_control` is set for which context:

| context          | `ac1` | `ac2` | `ac3` | `ac4` |
|------------------|:-----:|:-----:|:-----:|:-----:|
| `files`          | x     | x     |       |       |
| `spa`            | x     |       |       |       |
| `endpoint "foo"` | x     |       |       |       |
| `endpoint "bar"` | x     |       | x     | x     |

### `hosts` configuration example

Example configuration: `hosts = [ "localhost:9090", "api-stage.wao.io", "api.wao.io", "*:8081" ]`

The example configuration above makes Couper listen to port `:9090`, `:8081` and `8080`.

![hosts_example](./hosts_example.png)

In a second step Couper compares the host-header information with the configuration.
In case of mismatch a system error occurs (HTML error, status 500).

### Referencing and overwriting example

TBA
