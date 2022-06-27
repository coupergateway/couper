server {
  api {
    endpoint "/**" {
      proxy {
        backend = "rs"
      }
    }
  }
}

definitions {
  backend "rs" {
    origin = "{{ .origin }}"

    oauth2 {
      grant_type = "client_credentials"
      client_id = "cli"
      client_secret = "cls"
      token_endpoint = "{{ .token_endpoint }}/token"
      retries = 3
    }
  }
}
