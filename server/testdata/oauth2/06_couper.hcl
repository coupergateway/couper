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
    client_id = "foo"
    client_secret = "etbinbp4in"
    csrf {
      token_param = "state"
      token_value = request.cookies.st
    }
  }
}
