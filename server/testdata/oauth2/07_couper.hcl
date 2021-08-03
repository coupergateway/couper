server "client" {
  api {
    endpoint "/cb" {
      access_control = ["ac"]
      response {
        json_body = request.context.ac
      }
    }
  }
}

definitions {
  beta_oidc "ac" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    ttl = "1h"
    client_id = "foo"
    client_secret = "etbinbp4in"
    redirect_uri = "http://localhost:8080/cb" # value is not checked
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.cookies.nnc
  }
}
