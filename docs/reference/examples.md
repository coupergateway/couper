# Configuration Reference ~ Examples

* [Exposing APIs](#exposing-apis)
* [File and Web Serving](#file-and-web-serving)
* [Path Routing and Mapping](#path-routing-and-mapping)
* [Securing APIs](#securing-apis)
* [Variables and Expressions](#variables-and-expressions)

## Exposing APIs

Couper's main concept is exposing APIs via the configuration [Endpoint Block](blocks/endpoint.md),
fetching data from upstream or remote services, represented by the configuration
[Backend Block](blocks/backend.md).

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

This basic configuration defines an upstream backend service (`https://httpbin.org`)
and "mounts" it on the local API endpoint `/public/**`.

An incoming request `/public/anything` will result in an outgoing request
`https://httpbin.org/anything`.

## File and Web Serving

Couper contains a Web server for simple file serving and also takes care of the
more complex WEB serving of SPA assets.

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

The [Files Block](blocks/files.md) configures Couper's file server. The
[`document_root`](attributes.md) attribute configures the directory to serve.

The [SPA Block](blocks/spa.md) is responsible for serving the bootstrap document
configured via the [`bootstrap_file`](attributes.md) attribute for all paths that
match the [`paths`](attributes.md) list.

## Securing APIs

The following configuration protects the [Endpoint Block](blocks/endpoint.md)
`/private/**`. The [`access_control`](attributes.md) attribute in the `endpoint`
references the [JWT Block](blocks/jwt.md) defined in the [Definitions Block](blocks/definitions.md)
via the "accessToken" `label`.

```hcl
server "example" {
  endpoint "/private/**" {
    access_control = ["accessToken"]
    path = "/**"

    proxy {
      url = "https://httpbin.org"
    }
  }
}

definitions {
  jwt "accessToken" {
    header = "Authorization"
    key_file = "keys/public.pem"
    signature_algorithm = "RS256"
  }
}
```

## Path Routing and Mapping

```hcl
server "example" {
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
  }

  endpoint "/account/{id}" {
    proxy {
      backend {
        path = "/user/${request.path_params.id}/info"
        origin = "http://accountservice:8080"
      }
    }
  }
}
```

| Incoming request         | Outgoing request                              |
| ------------------------ | --------------------------------------------- |
| `/api/v1/login/foo`      | `http://identityprovider:8080/login/foo`      |
| `/api/v1/cart/items`     | `http://cartservice:8080/api/v1/items`        |
| `/api/v1/account/brenda` | `http://accountservice:8080/user/brenda/info` |

## Variables and Expressions

An example to send additional HTTP header fields to a configured backend and gets
evaluated on per-request basis.

```hcl
server "example" {
  endpoint "/" {
    proxy {
      backend {
        origin = env.ORIGIN

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

defaults {
  environment_variables = {
    ORIGIN = "https://example.com"
  }
}
```

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Error Handling](error-handling.md)
* &#8674; [Functions](functions.md)
