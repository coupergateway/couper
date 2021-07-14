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
    redirect_uri = "http://localhost:8080/cb" # value is not checked
    issuer = "https://authorization.server"
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    scope = "openid profile email"
    token_endpoint = "{{.asOrigin}}/token"
    userinfo_endpoint = "{{.asOrigin}}/userinfo"
    client_id = "foo"
    client_secret = "etbinbp4in"
    csrf {
      token_param = "nonce"
      token_value = request.cookies.nnc
    }
  }
}
