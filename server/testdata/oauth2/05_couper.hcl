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
    token_endpoint_auth_method = "client_secret_post"
    client_id = "foo"
    client_secret = "etbinbp4in"
    error_handler {
      response {
        status = 400
      }
    }
  }
}
