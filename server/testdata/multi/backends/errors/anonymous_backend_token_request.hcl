server {
  endpoint "/" {
    proxy {
      backend {
        origin = "https://example.com"
        beta_token_request {
          url = "https://as/token"
          backend {
            origin = "https://as"
          }
          backend = "BE"
        }
      }
    }
  }
  backend "BE" {
  }
}
