# Couper Docs - Version 0.1

## Table of contents

* [Introduction](#introduction)
  * [Core concepts](#core_concepts)
  * [Configuration file](#conf_file)
     * [Basic file structure](#basic_conf)
     * [Variables](#variables_conf)
     * [Expressions](#expressions)
     * [Functions](#functions)
* [Reference](#reference) 
  * [The `server` block](#server_block)
  * [The `files` block](#files_block)
  * [The `spa` block](#spa_block) 
  * [The `api` block](#api_block) 	
  * [The `endpoint` block](#endpoint_block)
  * [The `backend` block](#backend_block)
  * [The `request` block](#request_block) 
  * [The `cors` block](#cors_block)
  * [The `access_control` attribute](#access_control_attribute)   
  * [The `basic_auth` block](#basic_auth_block)
  * [The `jwt` block](#jwt_block)
  * [The `definitions` block](#definitions_block)
  * [The `defaults` block](#defaults_block)
  * [The `settings` block](#settings_block)     
* [Examples](#examples)
  * [Request routing](#request_routing_ex)
  * [Routing configuration](#routing_conf_ex)
  * [Web serving configuration](#web_serving_ex)
  * [`access_control`configuration](#access_control_conf_ex)
  * [`hosts` configuration](#hosts_conf_ex) 
   
## Introduction <a name="introduction"></a>
Couper is a frontend gateway especially designed to support building and running API-driven Web projects.
Acting as a proxy component it connects clients with (micro) services and adds access control and observability to the project. Couper does not need any special development skills and offers easy configuration and integration. 

## Core concepts <a name="core_concepts"></a>

![](./overview.png)

| Concept / Feature | Description                           |
|:-------------------|:---------------------------------------|
| Client(s)| Browser, App or API Client that sends requests to Couper.|
| Web Serving|Couper supports file serving and Web serving for SPA assets.|
|API| Configuration block that bundles endpoints under a certain base path.|
|Access Control| Couper handles access control for incoming client requests.|
|Endpoint| Configuration block that specifies how (and if) requests are sent to backend service(s) after they reach Couper.|
|Backend|Configuration block that specifies the connection to a local/remote backend service.|
|Logging|Couper provides standard logs for analysis and monitoring.|
|Backend Service(s)|External API or micro services where Couper fetches data from.|

## Configuration file <a name="conf_file"></a>

The syntax for Couper's configuration file is [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax), a configuration language by HashiCorp. 

### Basic file structure <a name="basic_conf"></a>
Couper's configuration file consists of nested configuration blocks that configure web serving and routing of the gateway. Access control is controlled by an `access_control` attribute that can be set for blocks. 

For orientation compare the fallowing example and the information below:

```hcl
server "my_project" {		
	files {...}
	spa {...}
	api {
		access_control = "foo"
		endpoint "/bar" {
			backend {...}
		}
	}
definitions {...}
```

* `server`: main configuration block
* `files`: configuration block for file serving
* `spa`: configuration block for web serving (spa assets) 
* `api`: configuration block that bundles endpoints under a certain base path
* `access_control`: attribute that sets access control for a block context
* `endpoint`: configuration block for Couper's entry points
* `backend`: configuration block for connection to local/remote backend service(s)
* `definitions`: block for predefined configurations, that can be referenced
* `defaults`: block for default configurations
* `settings`: block for server configuration which applies to the running instance

### Variables <a name="variables_conf"></a>

The configuration file allows the use of some predefined variables. There are two phases when those variables get evaluated.
The first phase is at config load which is currently related to `env` and **function** usage.
The second evaluation will happen during the request/response handling.

* `env` are the environment variables
* `req` is the client request
* `bereq` is the modified backend request
* `beresp` is the original backend response

Most fields are self-explanatory (compare tables below).

#### `env` variables
| Variable | Description                           |
|:-------------------|:-------------------------------|
|tba|tba|
|...|...|

#### `req` (client request) variables

| Variable | Description                           |
|:-------------------|:-------------------------------|
|`id` | unique request id |
| `method` | HTTP method|
| `path` | URL path|
| `endpoint` | matched endpoint pattern
| `headers.<name>` | HTTP request header value for requested lower-case key|
| `cookies.<name>` | value from `Cookie` request header for requested key (&#9888; last wins!)|
| `query.<name>` | query parameter values (&#9888; last wins!)|
| `post.<name>` | post form parameter|
| `ctx.<name>.<claim_name>` | request context containing claims from JWT used for [access control](#access_control_attribute), `<name>` being the [`jwt` block's](#jwt_block) label and `claim_name` being the claim's name|

#### `bereq`(modified backend request) variables 

| Variable | Description                           |
|:-------------------|:-------------------------------|
|`id` | unique request id |
| `method` | HTTP method|
| `path` | URL path|
| `headers.<name>` | HTTP request header value for requested lower-case key|
| `cookies.<name>` | value from `Cookie` request header for requested key (&#9888; last wins!)|
| `query.<name>` | query parameter values (&#9888; last wins!)|
| `post.<name>` | post form parameter|
| `ctx.<name>.<claim_name>` | request context containing claims from JWT used for [access control](#access_control_attribute), `<name>` being the [`jwt` block's](#jwt_block) label and `claim_name` being the claim's name|
|`url`|backend origin URL|

#### `beresp` (original backend response) variables
| Variable | Description                           |
|:-------------------|:-------------------------------|
| `status` | HTTP status code |
| `headers.<name>` | HTTP response header value for requested lower-case key |
| `cookies.<name>` | value from `Set-Cookie` response header for requested key (&#9888; last wins!)|

##### Variable Example

An example to send an additional header with client request header to a configured backend and gets evaluated on per request basis:

```hcl
server "variables-srv" {
  api {
    endpoint "/" {
      backend "my_backend_definition" {
        request_headers = {
          x-env-user = env.USER
          user-agent = "myproxyClient/${req.headers.app-version}"
          x-uuid = req.id
        }
      }
    }
  }
}
```

### Expressions <a name="expressions">
Since we use HCL2 for our configuration, we are able to use attribute values as expression:

```hcl
# Arithmetic with literals and application-provided variables
sum = 1 + addend

# String interpolation and templates
message = "Hello, ${name}!"

# Application-provided functions
shouty_message = upper(message)
```

### Functions <a name="functions">

Functions are little helper methods which are registered for every hcl evaluation context.

- `base64_decode`
- `base64_encode`
- `to_upper`
- `to_lower`

Example usage:

```hcl
my_attribute = base64_decode("aGVsbG8gd29ybGQK")
```

## Reference <a name="reference"></a>

### The `server` block <a name="server_block"></a> 
The `server` block is the main configuration block of Couper's configuration file.
It has an optional label and a `hosts` attribute. Nested blocks are `files`, `spa` and `api`. You can declare `access_control` for the `server` block. `access_control` is inherited by nested blocks.

| Name | Description                           |
|:-------------------|:-------------------------------|
|context|none|
| *label*|optional|
| `hosts`|<ul><li>list  </li><li>&#9888; mandatory, if there is more than one `server` block</li><li>*example:*`hosts = ["example.com", "..."]`</li><li>you can add a specific port to your host <br> *example:* `hosts = ["localhost:9090"]` </li><li>default port is `8080`</li><li>only **one** `hosts` attribute per `server` block is allowed</li><li>compare the hosts [example](#hosts_conf_ex) for details</li></ul>|
|[**`access_control`**](#access_control_attribute)|<ul><li>sets predefined `access_control` for `server` block</li><li>*example:* `access_control = ["foo"]`</li><li>&#9888; inherited</li></ul>|
|[**`files`**](#fi) block|configures file serving|
|[**`spa`**](#spa) block|configures web serving for spa assets|
|[**`api`**](#api) block|configures routing and backend connection(s)|


### <a name="fi"></a> The `files` block <a name="files_block"></a>
The `files` block configures your document root and the location of your error document. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
| *label*|optional|
| `document_root`|<ul><li>location of the document root</li><li>*example:* `document_root = "./htdocs"`</li></ul>|
|`error_file`|<ul><li>location of the error file</li><li>*example:* `error_file = "./404.html" `</li></ul>|
|[**`access_control`**](#access_control_attribute)|<ul><li>sets predefined `access_control` for `files` block context</li><li>*example:* `access_control = ["foo"]`</li></ul>|

### <a name="spa"></a>The `spa` block <a name="spa_block"></a>
The `spa` block configures the location of your bootstrap file and your SPA paths. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
| *label*|optional|
|`bootstrap_file`|<ul><li>location of the bootstrap file</li><li>*example:* `bootstrap_file = "./htdocs/index.html" "`</li></ul>|
|`paths`|<ul><li>list of SPA paths that need the bootstrap file</li><li>*example:* `paths = ["/app/**"]"`</li></ul>|
|[**`access_control`**](#access_control_attribute)|<ul><li>sets predefined `access_control` for `api` block context</li><li>*example:* `access_control = ["foo"]`</li></ul>|

### <a name="api"></a> The `api` block <a name="api_block"></a>
The `api` block contains all information about endpoints and the connection to remote/local backend service(s) (configured in the nested `endpoint` and `backend` blocks). You can add more than one `api` block to a `server` block.

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
|*label*|&#9888; mandatory, if there is more than one `api` block|
| `base_path`|<ul><li>optional</li><li>*example:* `base_path = "/api" `</li></ul>|
|[**`access_control`**](#access_control_attribute)|<ul><li>sets predefined `access_control` for `api` block context</li><li>&#9888; inherited by all endpoints in `api` block context</li></ul>|
|[**`backend`**](#backend_block) block|<ul><li>configures connection to a local/remote backend service for `api` block context</li><li>&#9888; only one `backend` block per `api` block<li>&#9888; inherited by all endpoints in `api` block context</li></ul>|
|[**`endpoint`**](#endpoint_block) block|configures specific endpoint for `api` block context|
|[**`cors`**](#cors_block) block|configures CORS behaviour for `api` block context|

### <a name="cors"></a> The `cors` block <a name="cors_block"></a>
The CORS block configures the CORS (Cross-Origin Resource Sharing) behaviour in Couper.

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`api`block|
| `allowed_origins` | (list of) allowed origin(s), can be either a string with a single specific origin (e.g. `https://www.example.com`) or `*` (all origins allowed) or an array of specific origins (`["https://www.example.com", "https://www.another.host.org"]`) |
| `allow_credentials = true` | if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` headers) |
| `max_age` | indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response headers can be cached (string with time unit, e.g. `"1h"`) |

### <a name="ep"></a> The `endpoint` block <a name="endpoint_block"></a>
Endpoints define the entry points of Couper. The mandatory *label* defines the path suffix for the incoming client request. The `path` attribute changes the path for the outgoing request (compare [request routing example](#request_routing_ex)). Each `endpoint` must have at least one `backend` which can be declared in the `api` context above or inside an `endpoint`. 

| Name | Description                           |
|:-------------------|:--------------------------------------|
|context|`api` block|
|*label*|<ul><li>&#9888; mandatory</li><li>defines the path suffix for incoming client requests</li><li>*example:* `endpoint "/dashboard" { `</li><li>incoming client request: `example.com/api/dashboard`</li></ul>|
| `path`|<ul><li>changeable part of upstream URL</li><li>changes the path suffix of the outgoing request</li></ul>|
|[**`access_control`**](#access_control_attribute)|sets predefined `access_control` for `endpoint`|
|[**`backend`**](#backend_block) block |configures connection to a local/remote backend service for `endpoint`|

### <a name="be"></a> The `backend` block <a name="backend_block"></a>
A `backend` defines the connection to a local/remote backend service. Backends can be defined globally in the `api` block for all endpoints of an API or inside an `endpoint`. An `endpoint` must have (at least) one `backend`. You can also define backends in the `definitions` block and use the mandatory *label* as reference. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|<ul><li>`api` block</li><li>`endpoint` block</li><li>`definitions` block (reference purpose)</li></ul>|
| *label*|<ul><li>&#9888; mandatory, when declared in `api` block</li><li>&#9888; mandatory, when declared in `definitions` block</li></ul>|
| `origin`||
|`base_path`|<ul><li>`base_path` for backend</li><li>won\`t change for `endpoint`</li></ul> |
|`path`|changeable part of upstream URL|
|`timeout`||
|`max_parallel_requests`||
| `request_headers`||
|[**`request`**](#request_block) block|<ul><li>configures` request` to backend service</li><li>optional, otherwise proxy mode</li></ul>|

### <a name="rq"></a> The `request` block <a name="request_block"></a>
| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`backend` block|
| `url`||
| `method`||
| `headers`||

### <a name="ac"></a> The `access_control` attribute <a name="access_control_attribute"></a> 
The `access_control` attribute let you set different `access_control` types for parts of your gateway. It is a list element that holds labels of predefined `access_control` types. You can set `access_control` for a certain block by putting `access_control = ["foo"]` in the corresponding block (where `foo` is an `access_control` type predefined in the `definitions` block). `access_control` is allowed in all blocks of Couper's configuration file. &#9888; access rights are inherited by nested blocks. You can also disable `access_control` for blocks. By typing `disable_access_control = ["bar"]`, the `access_control` type `bar` will be disabled for the corresponding block context.

Compare the `access_control` [example](#access_control_conf_ex) for details. 

#### <a name="ba"></a> The `basic_auth` block <a name="basic_auth_block"></a>
The `basic_auth` block let you configure basic auth for your gateway. Like all `access_control` types, the `basic_auth` block is defined in the `definitions` block and can be referenced in all configuration blocks by its mandatory *label*. 

If both `user`/`password` and `htpasswd_file` are configured, the incoming credentials from the `Authorization` request header are checked against `user`/`password` if the user matches, and against the data in the file referenced by `htpasswd_file` otherwise.

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|<ul><li>`server` block</li><li>`files` block</li><li>`spa` block</li><li>`api` block</li><li>`endpoint` block</li></ul>|
|*label*|<ul><li>&#9888; mandatory</li><li>always defined in `definitions` block</li></ul>|
|`user`| The user name |
|`password`| The corresponding password |
|`htpasswd_file`| The htpasswd file |
|`realm`| The realm to be sent in a `WWW-Authenticate` response header |


#### <a name="jwt"></a> The `jwt` block <a name="jwt_block"></a>
The `jwt` block let you configure JSON Web Token access control for your gateway. Like all `access_control` types, the `jwt` block is defined in the `definitions` block and can be referenced in all configuration blocks by its mandatory *label*. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|<ul><li>`server` block</li><li>`files` block</li><li>`spa` block</li><li>`api` block</li><li>`endpoint` block</li></ul>|
|*label*|<ul><li>&#9888; mandatory</li><li>always defined in `definitions` block</li></ul>|
|`cookie = "AccessToken"`||
|`header = "Authorization`|&#9888; impliziert Bearer|
|`header = "API-Token`||
|`post_param = "token"`||
|`query_param = "token"`||
|`key`||
|`key_file`||
|`signature_algorithm`||
|**`claims`**|equals/in comparison with JWT payload|

### The `definitions` block <a name="definitions_block"></a>
Use the `definitions` block to define configurations you want to reuse. `access_control` is **always** defined in the `definitions` block.

### The `defaults` block <a name="defaults_block"></a>

### The `settings` block <a name="settings_block"></a>
The `settings` block let you configure the more basic and global behaviour of your gateway instance.

| Name | Description                           | Default |
|:-------------------|:---------------------------------------|:-----------|
|`health_path`| The health path which is available for all configured server and ports. | `/healthz` |
|`default_port`| The port which will be used if not explicitly specified per host within the [`hosts`](#server_block) list. | `8080` |
|`log_format`| Switch for tab/field based colored view or json log lines. | `common` |
|`xfh`| Option to use the `X-Forwarded-Host` header as the request host | `false` |

### Health-Check ###
The health check will answer a status `200 OK` on every port with the configured `health_path`.
As soon as the gateway instance will receive a `SIGINT` or `SIGTERM` the check will return a status `500 StatusInternalServerError`.
A shutdown delay of `5s` allows the server to finish all running requests and gives a load-balancer time to pick another gateway instance.
After this delay the server goes into shutdown mode with a deadline of `5s` and no new requests will be accepted.
The shutdown timings cannot be configured at this moment. 

## Examples <a name="examples"></a>

### Request routing example <a name="request_routing_ex"></a> 
![](./routing_example.png)

| No. | configuration source |
|:-------------------|:---------------------------------------|
|1|`hosts` attribute in `server` block |
|2|`base_path` attribute in `api` block|
|3|*label* of `endpoint` block|
|4|`origin` attribute in `backend` block|
|5|`base_path` attribute in `backend`|
|6|`path` attribute in `endpoint` or `backend` block|

### Routing configuration example <a name="routing_conf_ex"></a>

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
```

### Web serving configuration example <a name="web_serving_ex"></a> 
```hcl
server "my_project" {		
	files {
		document_root = "./htdocs"
		error_file = "./404.html"
	}
	spa {
		bootstrap_file = "./htdocs/index.html"
		paths = [
			"/app/**",
			"/profile/**"
		]
	}
...
```

### `access_control` configuration example <a name="access_control_conf_ex"></a> 

```hcl
server {
	access\_control = ["ac1"]
	files {
		access\_control = ["ac2"]
	}
	spa {}
	api {
		access\_control = ["ac3"]
		endpoint "/foo" {
			disable\_access_control = "ac3"
		}
		endpoint "/bar" {
		access\_control = ["ac4"]
		}
	}
}
definitions {
 	basic\_auth "ac1" {
 	...
 	}	
 	jwt "ac2" {
 	...
 	}
 	jwt "ac3" {
 	...
 	}
 	jwt "ac4" {
 	...
 	}
}
```

The following table shows which `access_control` is set for which context:

| context | `ac1`|`ac2`|`ac3`|`ac4`|
|----|:-----:|:---:|:---:|:---:|
|`files`|x|x|||
|`spa`|x||||
|`endpoint "foo"` |x||||
|`endpoint "bar"` |x||x|x|

### `hosts` configuration example <a name="hosts_conf_ex"></a> 
Example configuration: `hosts = [ "localhost:9090", "api-stage.wao.io", "api.wao.io", "*:8081" ]`
 
The example configuration above makes Couper listen to port `:9090`, `:8081`and `8080`. 
  
![](./hosts_example.png)

In a second step Couper compares the host-header information with the configuration. In case of mismatch a system error occures (HTML error, status 500).  

### Referencing and overwriting example

TBA