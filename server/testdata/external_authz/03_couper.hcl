server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        headers = {
          x-authz-sub = request.context.authz.sub
        }
        json_body = request.context.authz
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    endpoint "/check" {
      response {
        json_body = {
          sub   = "clark.kent"
          roles = ["reporter", "hero"]
        }
      }
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url = "http://127.0.0.1:8081/check"
  }
}
