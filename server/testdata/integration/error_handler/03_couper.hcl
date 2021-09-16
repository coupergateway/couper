server "access_control" {
  endpoint "/" {
    access_control = ["ba"]
    response {
      status = 204
    }
  }
}

definitions {
  basic_auth "ba" {
    password = "couper"

    error_handler {
      request "another" {
        url = "https://as/foo"
        backend = "as"
      }
      response {}
    }
  }
  backend "as" {
    origin = "https://as"
    oauth2 {
      # grant_type = "missing attribute!"
      token_endpoint = "https://as/token"
      client_id = "my-id"
      client_secret = "my-secret"
    }
  }
}
