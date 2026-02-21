server "logs" {
  api {
    endpoint "/cb" {
      access_control = ["ac"]
      response {
        status = 204
      }
    }
  }
}

definitions {
  oidc "ac" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    client_id = "foo"
    client_secret = "etbinbp4in"
    redirect_uri = "http://localhost:8080/cb"
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.cookies.nnc

    backend {
      origin = "{{.asOrigin}}"

      custom_log_fields = {
        token_type = backend_response.json_body.token_type
      }
    }
  }
}
