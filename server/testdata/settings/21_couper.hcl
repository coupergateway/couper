server {
  endpoint "/" {
    proxy {
      backend = "rs"
    }
  }
}

definitions {
  backend "rs" {
    oauth2 {
      token_endpoint = "http://as.com/token"
      backend = "as"
      grant_type = "client_credentials"
      client_id = "xyz"
      client_secret = "xyz"
    }
  }

  backend "as" {
    beta_token_request {
      url = "http://vault.com/key"
      token = "xyz"
      ttl = "1h"
      backend = "rs"
    }
  }
}
