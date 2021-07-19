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
  beta_oauth2 "ac" {
    grant_type = "authorization_code"
    redirect_uri = "http://localhost:8080/cb" # value is not checked
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    token_endpoint = "{{.asOrigin}}/token"
    token_endpoint_auth_method = "client_secret_post"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
    client_id = "foo"
    client_secret = "etbinbp4in"
    error_handler {
      response {
        status = 400
      }
    }
  }
}
