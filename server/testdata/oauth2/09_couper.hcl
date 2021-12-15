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
  oidc "ac" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    client_id = "foo"
    client_secret = "etbinbp4in"
    redirect_uri = "/cb" # value is not checked
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.cookies.nnc
  }
}

settings {
  accept_forwarded_url = [ "proto", "host" ]
}
