server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        headers = {
          x-authz-proto = request.context.authz.proto
        }
      }
    }
  }
}

definitions {
  beta_authz_external "authz" {
    backend {
      origin                         = "{{.origin}}"
      http2                          = true
      disable_certificate_validation = true
    }
  }
}
