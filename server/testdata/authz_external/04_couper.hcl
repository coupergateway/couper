server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      # The resolved identity from the callout response header is injected explicitly;
      # set_request_headers would overwrite a client-provided value the same way.
      response {
        headers = {
          x-identity = request.context.authz.headers["x-resolved-identity"]
          x-evil     = request.context.authz.headers["x-evil"]
        }
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    endpoint "/check" {
      response {
        headers = {
          content-type        = "application/json"
          x-resolved-identity = "clark.kent"
        }
        json_body = {
          sub = "clark.kent"
        }
      }
    }
  }
}

definitions {
  beta_authz_external "authz" {
    url = "http://127.0.0.1:8081/check"
  }
}
