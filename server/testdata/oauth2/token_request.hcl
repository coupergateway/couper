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
      Auth-2 = backends.be.tokens.default
      Auth-3 = backends.be.token
      Auth-4 = backend.tokens.tr1
      Auth-5 = backend.tokens.default
      Auth-6 = backend.token
    }

    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      backend = "as"
      client_id = "clid"
      client_secret = "cls"
      grant_type = "client_credentials"
    }

    beta_token_request "tr1" {
      url = "{{.asOrigin}}/token1"
      backend = "as"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "client_credentials"
      }
      token = token_response.json_body.access_token
      ttl = "${default(token_response.json_body.expires_in, 3600) * 0.9}s"
    }

    beta_token_request "default" {
      url = "{{.asOrigin}}/token2"
      backend = "as"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "password"
        username = "user"
        password = "asdf"
      }
      token = token_response.json_body.access_token
      ttl = "${default(token_response.json_body.expires_in, 3600) * 0.9}s"
    }
  }

  backend "as" {
    origin = "{{.asOrigin}}"
  }
}

settings {
  no_proxy_from_env = true
}
