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
      token = beta_token_response.json_body.access_token
      ttl = "no duration"
    }
  }

  backend "as" {
    origin = "{{.asOrigin}}"
  }
}
