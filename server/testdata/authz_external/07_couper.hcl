server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        status = 204
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    endpoint "/check" {
      response {
        status = 401
        headers = {
          www-authenticate = "Bearer resource_metadata=\"http://protected.example/.well-known/oauth-protected-resource/protected\""
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
