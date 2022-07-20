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
      Auth-1 = backends.be.beta_tokens.tr1
      Auth-2 = backends.be.beta_tokens.default
      Auth-3 = backends.be.beta_token
      Auth-4 = beta_backend.beta_tokens.tr1
      Auth-5 = beta_backend.beta_tokens.default
      Auth-6 = beta_backend.beta_token
      KeyId = backends.as.beta_token
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
      token = beta_token_response.json_body.access_token
      ttl = "${default(beta_token_response.json_body.expires_in, 3600) * 0.9}s"
    }

    beta_token_request "default" {
      url = "{{.asOrigin}}/token2"
      query_params = {
        foo = "bar"
      }
      backend = "as"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "password"
        username = "user"
        password = "asdf"
      }
      token = beta_token_response.json_body.access_token
      ttl = "${default(beta_token_response.json_body.expires_in, 3600) * 0.9}s"
    }
  }

  backend "as" {
    origin = "{{.asOrigin}}"
    set_request_headers = {
      KeyId = beta_backend.beta_token
    }

    beta_token_request {
      url = "{{.vaultOrigin}}/key"
      backend = "vault"
      token = beta_token_response.body
      ttl = "1h"
    }
  }

  backend "vault" {
    origin = "{{.vaultOrigin}}"
  }
}

settings {
  no_proxy_from_env = true
}
