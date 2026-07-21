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
  beta_external_authz "authz" {
    backend {
      origin                = "{{.origin}}"
      http2_prior_knowledge = true
    }
  }
}
