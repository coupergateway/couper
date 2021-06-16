* [Introduction](#introduction)
   * [Architectural Overview](#architectural-overview)
   * [Getting Started](#getting-started)
   * [Configuration File](#configuration-file)
      * [IDE Extension](#ide-extension)
      * [File Name](#file-name)
      * [Basic File Structure](#basic-file-structure)
      * [Expressions](#expressions)
         * [Variables](#variables)
      * [Examples](#examples)
         * [File &amp; Web Serving](#file-&-web-serving)
         * [Exposing APIs](#exposing-apis)
         * [Securing APIs](#securing-apis)
         * [Routing: Path Mapping](#routing:-path-mapping)
         * [Using Variables](#using-variables)

# Introduction

Couper is a lightweight open-source API gateway that acts as an entry point for clients to your application (frontend API gateway) and an exit point to upstream services (upstream API gateway).

It exposes endpoints with use cases and adds access control, observability, and back-end connectivity on a separate layer. This will keep your core application code more simple.

Couper does not need any special development skills and offers easy configuration and integration.

## Architectural Overview

![architecture](./architecture.png)

| Entity  | Description |
|:-------------------|:------------|
| Frontend         | Browser, App or API Client that sends requests to Couper. |
|Frontend API | Couper acts as an entry point for clients to your application. |
|Backend Service|Your core application - no matter which technology, if monolithic or micro-services.|
|Upstream API| Couper acts as an exit point to upstream services for your application. |
|Remote Service|Any upstream service or system which is accessible via HTTP. |
|Protected System| Representing a service or system that offers protected resources via HTTP.|

## Getting Started

Couper is available as _docker
image_ from [Docker Hub](https://hub.docker.com/r/avenga/couper)

Running Couper requires a working [Docker](https://www.docker.com/) setup on your
computer. Please visit the [get started guide](https://docs.docker.com/get-started/) to get prepared.

To download/install Couper, open a terminal and execute:

```sh
$ docker pull avenga/couper
```

Couper needs a configuration file to know what to do. 

Create a directory with an empty `couper.hcl` file. 

Copy/paste the following configuration to the file and save it. 

```hcl
server "hello" {
	endpoint "/**" {
        response {
            body = "Hello World!"   
        }
    }
}
```
Now `cd` into the directory with the configuration file and start Couper in a docker container:

```sh
$ docker run --rm -p 8080:8080 -v "$(pwd)":/conf avenga/couper

{"addr":"0.0.0.0:8080","level":"info","message":"couper gateway is serving","timestamp":"2020-08-27T16:39:18Z","type":"couper"}
```
Now Couper is serving on your computer's port *8080*. Point your
browser or `curl` to [`localhost:8080`](http://localhost:8080/) to see what's going on.

Press `CTRL+c` to stop the container.

## Configuration File
The language for Couper's configuration file is [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax), a configuration language by HashiCorp.

### IDE Extension
Couper provides its own IDE extension that adds Couper-specific highlighting and autocompletion to Couper's configuration file `couper.hcl` in Visual Studio Code.

Get it from the [Visual Studio Market Place](https://marketplace.visualstudio.com/items?itemName=AvengaGermanyGmbH.couper) or visit the [Extension repository](https://github.com/avenga/couper-vscode).


### File Name

The file-ending of your configuration file should be `.hcl` to have syntax highlighting within your IDE.

The file name defaults to `couper.hcl` in your working directory. This can be changed with the `-f` command-line flag. With `-f /opt/couper/my_conf.hcl` couper changes the working directory to `/opt/couper` and loads `my_conf.hcl`.

### Basic File Structure

Couper's configuration file consists of nested configuration blocks that configure
the gateway. There are a large number of options, but let's focus on the main structure first:

```hcl
server "my_project" {
  files { 
    ...
  }

  spa {
    ...
  }

  api {
    access_control = ["foo"]
    endpoint "/bar" {
      proxy {
        backend {...}
      }
      request "sub-request" {
        backend {...}
      }
      response {...}
    }
  }
}

definitions {
  ...
}

settings {
  ...
}
```

* `server` main configuration block(s)
  * `files` configuration block for file serving
  * `spa` configuration block for Web serving (SPA assets)
  * `api` configuration block(s) that bundle(s) endpoints under a certain base path or `access_control` list
  * `access_control` attribute that sets access control for a block context
  * `endpoint` configuration block for Couper's entry points
    * `proxy` configuration block for a proxy request and response to an origin
      * `backend` configuration block for connection to local/remote backend service(s)
    * `request` configuration block for a manual request to an origin
      * `backend` configuration block for connection to local/remote backend service(s)
    * `response` configuration block for a manual client response
* `definitions` block for predefined configurations, that can be referenced
* `settings` block for server configuration which applies to the running instance

### Expressions

Since we use [HCL 2.0](https://github.com/hashicorp/hcl/tree/hcl2#information-model-and-syntax) for our configuration, we are able to use attribute values as expression.

```hcl
// Arithmetic with literals and application-provided variables
sum = 1 + addend

// String interpolation and templates
message = "Hello, ${name}!"

// Application-provided functions
shouty_message = upper(message)
```
See [function reference](./REFERENCE.md#functions).

#### Variables

The configuration file allows the use of some predefined variables. There are two phases when those variables get evaluated.
The first phase is at config load which is currently related to `env` and simple **function** usage.
The second evaluation will happen during the request/response handling.

* `env` are the environment variables
* `request` is the client request
* `backend_requests` contains all modified backend requests
* `backend_responses` contains all original backend responses

See [variables reference](./REFERENCE.md#variables) for details.

### Examples

#### File & Web Serving 
Couper contains a Web server for simple file serving and also takes care of the more complex web serving of SPA assets.

```hcl
server "example" {

  files {
    document_root = "htdocs"
  }

  spa {
    bootstrap_file = "htdocs/index.html"
    paths = ["/**"]
  }
}
```
The `files` block configures Couper's file server. It needs to know which directory to serve (`document_root`).

The `spa` block is responsible for serving the bootstrap document for all paths that match the paths list.

#### Exposing APIs

Couper's main concept is exposing APIs via the configuration block `endpoint`, fetching data from upstream or remote services, represented by the configuration block `backend`.

```hcl
server "example"{

  endpoint "/public/**"{
    path = "/**"
  
    proxy {
      backend {
        origin = "https://httpbin.org"
      }
    }
  }
}
```
This basic configuration defines an upstream backend service (`https://httpbin.org`) and "mounts" it on the local API endpoint `/public/**`. 

An incoming request `/public/foo` will result in an outgoing request `https://httpbin.org/foo`.

#### Securing APIs
Access control is controlled by an
[Access Control](./REFERENCE.md#access-control) attribute that can be set for many blocks.

```hcl
server "example" {

  endpoint "/private/**" {
    access_control = ["accessToken"]
    path = "/**"

    proxy {
      backend {
        origin = "https://httpbin.org"
      }
    }
  }
    
  definitions {
    jwt "accessToken" {
      signature_algorithm = "RS256"
      key_file = "keys/public.pem"
      header = "Authorization"
    }
  }
}
```
This configuration protects the endpoint `/private/**` with the access control `"accessToken"` configured in the `definitions` block. 

#### Routing: Path Mapping
```hcl
api "my_api" {
  base_path = "/api/v1"
  
  endpoint "/login/**" {
 
    proxy {
      backend {
        origin = "http://identityprovider:8080"
      }
    }
  }

  endpoint "/cart/**" {
     
      path = "/api/v1/**"
      proxy {
        url = "http://cartservice:8080"
      }

  endpoint "/account/{id}" {
    proxy {
      backend {
        path = "/user/${request.param.id}/info"
        origin = "http://accountservice:8080"
        }
    }
  }
}
```
| Incoming request   | Outgoing request |
|:--------------------------|:------------|
|/api/v1/login/foo|http://identityprovider:8080/login/foo|
|/api/v1/cart/items|http://cartservice:8080/api/v1/items|
|/api/v1/account/brenda|http://accountservice:8080/user/brenda/info|

#### Using Variables & Expressions

An example to send an additional header with client request header to a configured
backend and gets evaluated on per-request basis:

```hcl
server "variables-srv" {
  endpoint "/" {
    proxy {
      backend "my_backend_definition" {
        set_request_headers = {
          # simple variable lookup
          x-uuid = request.id
          # template string
          user-agent = "myproxyClient/${request.headers.app-version}"
          # expressions and function calls
          x-env-user = env.USER != "" ? upper(env.USER) : "UNKNOWN"
        }
      }
    }
  }
}
```
