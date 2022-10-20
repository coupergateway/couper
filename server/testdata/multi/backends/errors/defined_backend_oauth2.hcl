server {}
definitions {
  backend "b" {
    oauth2 {
      grant_type = "client_credentials"
      client_id = "cli"
      client_secret = "cls"
      token_endpoint = "https://as/token"
      backend {
        origin = "https://as"
      }
      backend = "BE"
    }
  }
  backend "BE" {
  }
}
