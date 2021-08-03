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
    redirect_uri = "/cb" # value is not checked
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    token_endpoint = "{{.asOrigin}}/token"
    client_id = "foo"
    client_secret = "etbinbp4in"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
  }
}

settings {
  accept_forwarded_url = [ "proto", "host" ]
}
