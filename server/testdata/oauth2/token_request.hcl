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
    origin = "{{.rsOrigin}}"
    set_request_headers = {
      Auth-1 = backends.be.tokens.tr1
      Auth-2 = backends.be.tokens.tr2
    }

    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      client_id = "clid"
      client_secret = "cls"
      grant_type = "client_credentials"
    }

    beta_token_request "tr1" {
      url = "{{.asOrigin}}/token1"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "client_credentials"
      }
      token = token_response.json_body.access_token
      ttl = "${default(token_response.json_body.expires_in, 3600) * 0.9}s"
    }

    beta_token_request "tr2" {
      url = "{{.asOrigin}}/token2"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "password"
        username = "user"
        password = "asdf"
      }
      token = token_response.body
      ttl = "2m"
    }
  }
}

settings {
  no_proxy_from_env = true
}
