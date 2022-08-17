server "api" {
  api {
    endpoint "/resource" {
      proxy {
        backend = "be"
      }

      error_handler "beta_backend_token_request" {
        response {
          status = 204
        }
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "http://does.not.matter"

    beta_token_request "tr1" {
      url = "/token1"
      backend = "down"
      form_body = {
        client_id = "clid"
        client_secret = "cls"
        grant_type = "client_credentials"
      }
      token = beta_token_response.json_body.access_token
      ttl = "${default(beta_token_response.json_body.expires_in, 3600) * 0.9}s"
    }

    beta_token_request "default" {
      url = "/token2"
      query_params = {
        foo = "bar"
      }
      backend = "down"
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

  backend "down" {
    origin = "http://1.2.3.4"
  }
}

settings {
  no_proxy_from_env = true
}
