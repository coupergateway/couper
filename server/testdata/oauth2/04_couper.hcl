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
  oauth2 "ac" {
    grant_type = "authorization_code"
    redirect_uri = "http://localhost:8080/cb" # value is not checked
    token_endpoint = "${request.headers.x-token-url}/token"
    client_id = "foo"
    client_secret = "etbinbp4in"
    code_verifier_value = request.cookies.pkcecv
  }
}
