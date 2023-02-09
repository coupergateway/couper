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
    configuration_url = "{{.asOrigin}}/without/userinfo/.well-known/openid-configuration"
    configuration_ttl = "1h"
    client_id = "foo"
    client_secret = "etbinbp4in"
    redirect_uri = "http://www.example.com/cb" # value is not checked
    verifier_value = request.cookies.nnc
  }
}
