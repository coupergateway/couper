server {
  api {
    endpoint "/resource" {
      proxy {
        backend = "be"
      }
    }
    endpoint "/other/resource" {
      proxy {
        backend = "be2"
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "http://does.not.matter"

    oauth2 {
      token_endpoint = "http://1.2.3.4/token1"
      backend = "as_down"
      client_id = "my_clid"
      grant_type = "urn:ietf:params:oauth:grant-type:jwt-bearer"
      assertion = "a jwt string"
    }
  }

  backend "be2" {
    origin = "http://does.not.matter"

    oauth2 {
      token_endpoint = "{{.asOrigin}}/token/error"
      client_id = "my_clid"
      client_secret = "my_cls"
      grant_type = "client_credentials"
    }
  }

  backend "as_down" {
    origin = "http://1.2.3.4"
  }
}
