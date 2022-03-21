server {
  api {
    base_path = "/api1"

    cors {
      allowed_origins = ["*"]
    }

    endpoint "/unrestricted" {
      access_control = ["token"]
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
      access_control = ["token"]
      allowed_methods = ["GET", "Post", "delete", "BREW"]
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

    endpoint "/wildcard" {
      allowed_methods = ["*"]

      response {
        body = "a"
      }
    }

    endpoint "/wildcardAndMore" {
      allowed_methods = ["get", "*", "PuT", "brew"]

      response {
        body = "a"
      }
    }

    endpoint "/blocked" {
      allowed_methods = []

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
