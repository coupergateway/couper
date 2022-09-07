server {
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

    oauth2 {
      token_endpoint = "{{.asOrigin}}/token1"
      client_id = "my_clid"
      grant_type = "urn:ietf:params:oauth:grant-type:jwt-bearer"
      assertion = []
    }
  }
}
