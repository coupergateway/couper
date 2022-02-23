server {
  api {
    base_path = "/api1"
    access_control = ["token"]

    cors {
      allowed_origins = ["*"]
    }

    endpoint "/unrestricted" {
      beta_scope = {
        get = "foo"
        head = "foo"
        post = "foo"
        put = "foo"
        patch = "foo"
        # no delete, no options
        brew = "foo"
      }
      response {
        body = "a"
      }
    }
  }
}

definitions {
  jwt "token" {
    signature_algorithm = "HS256"
    key = "asdf"
    beta_scope_claim = "scope"
  }
}
