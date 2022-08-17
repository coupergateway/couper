server "api" {
  api {
    endpoint "/resource" {
      proxy {
        backend = "be"
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "http://does.not.matter"

    beta_token_request "tr" {
      url = "/token"
      backend = "as"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "client_credentials"
      }
      token = []
      ttl = "${default(beta_token_response.json_body.expires_in, 3600) * 0.9}s"
    }
  }

  backend "as" {
    origin = "{{.asOrigin}}"
  }
}
