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
        # no delete, no options, no trace
        brew = "foo"
      }
      response {
        body = "a"
      }
    }

    endpoint "/restricted" {
      allowed_methods = ["GET", "Post", "delete", "tRaCe", "BREW"]
      beta_scope = {
        get = "foo"
        head = "foo"
        post = "foo"
        put = "foo"
        patch = "foo"
        # no delete, no options, no trace
        brew = "foo"
      }

      response {
        body = "a"
      }
    }
  }

  api {
    base_path = "/api2"
    allowed_methods = ["PUT"]

    endpoint "/restricted" {
      allowed_methods = ["GET", "Post", "delete", "BREW"]

      response {
        body = "a"
      }
    }

    endpoint "/restrictedByApiOnly" {
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
