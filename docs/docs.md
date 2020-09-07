# Couper 2.0 Docs - Version 0.1

<!--
* TODO: update access control in overview image  
* OPEN: allow `access_control` for backend????
* TODO: hosts attribute section needs some love
* TODO: defaults block 
* TODO: update basic auth section 
* TODO: better access_control example 
--->

## Introduction to Couper 2.0
Couper 2.0 is a lightweight API gateway especially designed to support building and running API-driven Web projects.
Acting as a proxy component it connects clients with (micro) services and adds access control and observability to the project. Couper does not need any special development skills and offers easy configuration and integration. 

## Core Concepts

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

## Configuration File

The syntax for Couper's configuration file is [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax), a configuration language by HashiCorp. 


---

### File structure
Couper's configuration file consists of nested configuration blocks that configure web serving and routing of the gateway. Access control is controlled by an `access_control` attribute that can be set for blocks. 

For orientation compare the fallowing example and the information below:

<pre><code>server "my_project" {		
	files {...}
	spa {...}
	api {
		access_control = "foo"
		endpoint "/bar" {
			backend {...}
		}
	}
definitions {...}
</code></pre>

* `server`: main configuration block
* `files`: configuration block for file serving
* `spa`: configuration block for web serving (spa assets) 
* `api`: configuration block that bundles endpoints under a certain base path
* `access_control`: attribute that sets access control for a block context
* `endpoint`: configuration block for Couper's entry points
* `backend`: configuration block for connection to local/remote backend service(s)
* `definitions`: block for predefined configurations, that can be referenced
* `defaults`: block for default configurations


### The `server` block 
The `server` block is the main configuration block of Couper's configuration file.
It has an optional label and a `hosts` attribute. Nested blocks are `files`, `spa` and `api`. You can declare `access_control` for the `server` block. `access_control` is inherited by nested blocks.

| Name | Description                           |
|:-------------------|:-------------------------------|
|context|none|
| *label*|optional|
| `hosts`|<ul><li>list  </li><li>*example:*`hosts = ["example.com", "..."]`</li><li>&#9888; mandatory, if there is more than one `server` block</li></ul>|
|[**`access_control`**](#ac)|<ul><li>sets predefined `access_control` for `server` block</li><li>*example:* `access_control = ["foo"]`</li><li>&#9888; inherited</li></ul>|
|[**`files`**](#fi) block|configures file serving|
|[**`spa`**](#spa) block|configures web serving for spa assets|
|[**`api`**](#api) block|configures routing and backend connection(s)|

### <a name="hosts"></a> The `hosts` attribute  
The `hosts` attribute specifies the requests your server should listen to. 

* default listen port: 8080
* if there is only one `server` block, the `hosts` attribute is optional
  * your server responds to **all** requests 
  * `hosts = ["*"]`
* if there is more than one `server` block, the `hosts` attribute is mandatory
* only **one** `hosts` attribute per `server` block is allowed


Compare the `hosts` [example](#example_hosts) for details. 


### <a name="fi"></a> The `files` block
The `files` block configures your document root and the location of your error document. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
| *label*|optional|
| `document_root`|<ul><li>location of the document root</li><li>*example:* `document_root = "./htdocs"`</li></ul>|
|`error_file`|<ul><li>location of the error file</li><li>*example:* `error_file = "./404.html" `</li></ul>|
|[**`access_control`**](#ac)|<ul><li>sets predefined `access_control` for `files` block context</li><li>*example:* `access_control = ["foo"]`</li></ul>|

### <a name="spa"></a>The `spa` block
The `spa` block configures the location of your bootstrap file and your SPA paths. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
| *label*|optional|
|`bootstrap_file`|<ul><li>location of the bootstrap file</li><li>*example:* `bootstrap_file = "./htdocs/index.html" "`</li></ul>|
|`paths`|<ul><li>list of SPA paths that need the bootstrap file</li><li>*example:* `paths = ["/app/**"]"`</li></ul>|
|[**`access_control`**](#ac)|<ul><li>sets predefined `access_control` for `api` block context</li><li>*example:* `access_control = ["foo"]`</li></ul>|

### <a name="api"></a> The `api` block
The `api` block contains all information about endpoints and the connection to remote/local backend service(s) (configured in the nested `endpoint` and `backend` blocks). You can add more than one `api` block to a `server` block.

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`server` block|
|*label*|&#9888; mandatory, if there is more than one `api` block|
| `base_path`|<ul><li>optional</li><li>*example:* `base_path = "/api" `</li></ul>|
|[**`access_control`**](#ac)|<ul><li>sets predefined `access_control` for `api` block context</li><li>&#9888; inherited by all endpoints in `api` block context</li></ul>|
|[**`backend`**](#be) block|<ul><li>configures connection to a local/remote backend service for `api` block context</li><li>&#9888; only one `backend` block per `api` block<li>&#9888; inherited by all endpoints in `api` block context</li></ul>|
|[**`endpoint`**](#ep) block|configures specific endpoint for `api` block context|


### <a name="ep"></a> The `endpoint` block
Endpoints define the entry points of Couper. The mandatory *label* defines the path suffix for the incoming client request. The `path` attribute changes the path for the outgoing request (compare [request routing example](#request_example)). Each `endpoint` must have at least one `backend` which can be declared in the `api` context above or inside an `endpoint`. 

| Name | Description                           |
|:-------------------|:--------------------------------------|
|context|`api` block|
|*label*|<ul><li>&#9888; mandatory</li><li>defines the path suffix for incoming client requests</li><li>*example:* `endpoint "/dashboard" { `</li><li>incoming client request: `example.com/api/dashboard`</li></ul>|
| `path`|<ul><li>changeable part of upstream url</li><li>changes the path suffix of the outgoing request</li></ul>|
|[**`access_control`**](#ac)|sets predefined `access_control` for `endpoint`|
|[**`backend`**](#be) block |configures connection to a local/remote backend service for `endpoint`|

### <a name="be"></a> The `backend` block
A `backend` defines the connection to a local/remote backend service. Backends can be defined globally in the `api` block for all endpoints of an API or inside an `endpoint`. An `endpoint` must have (at least) one `backend`. You can also define backends in the `definitions` block and use the mandatory *label* as reference. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|<ul><li>`api` block</li><li>`endpoint` block</li><li>`definitions` block (reference purpose)</li></ul>|
| *label*|<ul><li>&#9888; mandatory, when declared in `api` block</li><li>&#9888; mandatory, when declared in `definitions` block</li></ul>|
| `origin`||
|`base_path`|<ul><li>`base_path` for backend</li><li>won\`t change for `endpoint`</li></ul> |
|`path`|changeable part of upstream url|
|`timeout`||
|`max_parallel_requests`||
| `request_headers`||
|[**`request`**](#rq) block|<ul><li>configures` request` to backend service</li><li>optional, otherwise proxy mode</li></ul>|

#### <a name="rq"></a> The `request` block
| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|`backend` block|
| `url`||
| `method`||
| `headers`||

### <a name="ac"></a> The `access_control` attribute  
The `access_control` attribute let you set different `access_control` types for parts of your gateway. It is a list element that holds labels of predefined `access_control` types. You can set `access_control` for a certain block by putting `access_control = ["foo"]` in the corresponding block (where `foo` is an `access_control` type predefined in the `definitions` block). `access_control` is allowed in all blocks of Couper's configuration file. &#9888; access rights are inherited by nested blocks. You can also disable `access_control` for blocks. By typing `disable_access_control = ["bar"]`, the `access_control` type `bar` will be disabled for the corresponding block context.

Compare the `access_control` [example](#example_ac) for details. 

#### <a name="ba"></a> The `basic_auth` block
The `basic_auth` block let you configure basic auth for your gateway. Like all `access_control` types, the `basic_auth` block is defined in the `definitions` block and can be referenced in all configuration blocks by its mandatory *label*. 

| Name | Description                           |
|:-------------------|:---------------------------------------|
|context|<ul><li>`server` block</li><li>`files` block</li><li>`spa` block</li><li>`api` block</li><li>`endpoint` block</li></ul>|
|*label*|<ul><li>&#9888; mandatory</li><li>always defined in `definitions` block</li></ul>|
|`user`||
|`password`||
|`credentials`||
|`values`||

#### <a name="jwt"></a> The `jwt` block
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

### The `definitions` block 
Use the `definitions` block to define configurations you want to reuse. `access_control` is **always** defined in the `definitions` block.

### The `defaults` block 

#### Referencing and Overwriting

## Examples

### <a name="request_example"></a>Example 1: request routing
![](./routing_example.png)

| No. | configuration source |
|:-------------------|:---------------------------------------|
|1|`hosts` attribute in `server` block |
|2|`base_path` attribute in `api` block|
|3|*label* of `endpoint` block|
|4|`origin` attribute in `backend` block|
|5|`base_path` attribute in `backend`|
|6|`path` attribute in `endpoint` or `backend` block|

### Example 2: routing configuration

<pre><code>api "my_api" {
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
</code></pre>

### <a name="example_web_serving"></a> Example 3: Web serving configuration 

<pre><code>server "my_project" {		
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
</code></pre>


### <a name="example_ac"></a> Example 4: `access_control` configuration

<pre><code>server {
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
}</pre></code>

The following table shows which `access_control` is set for which context:

| context | `ac1`|`ac2`|`ac3`|`ac4`|
|----|:-----:|:---:|:---:|:---:|
|`files`|x|x|||
|`spa`|x||||
|`endpoint "foo"` |x||||
|`endpoint "bar"` |x||x|x|

### <a name="example_hosts"></a> Example 5: `hosts` configuration
Example configuration: `hosts = [ "localhost:9090", "api-stage.wao.io", "api.wao.io", "*:8081" ]`
 
The example configuration above makes Couper listen to port `:9090`, `:8081`and `8080`. 
  
![](./hosts_example.png)

In a second step Couper compares the host-header information with the configuration. In case of mismatch a system error occures (HTML error, status 500).  

