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
    issuer = "https://authorization.server"
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    scope = "openid profile email"
    token_endpoint = "${request.headers.x-as-origin}/token"
    userinfo_endpoint = "${request.headers.x-as-origin}/userinfo"
    client_id = "foo"
    client_secret = "etbinbp4in"
    csrf_token_param = "nonce"
    csrf_token_value = request.cookies.nnc
  }
}
