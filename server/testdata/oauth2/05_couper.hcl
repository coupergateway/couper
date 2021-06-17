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
    token_endpoint = "${request.headers.x-as-origin}/token"
    token_endpoint_auth_method = "client_secret_post"
    pkce {
      code_challenge_method = "S256"
      code_verifier_value = request.cookies.pkcecv
    }
    client_id = "foo"
    client_secret = "etbinbp4in"
    error_handler {
      response {
        status = 400
      }
    }
  }
}
