server {
  files {
    document_root = "../files/htdocs_a"

    cors {
      allowed_origins = ["*"]
    }
  }

  spa {
    bootstrap_file = "../files/htdocs_a/index.html"
    paths = ["/app/**"]

    cors {
      allowed_origins = ["*"]
    }
  }

  api {
    base_path = "/api1"

    cors {
      allowed_origins = ["*"]
    }

    endpoint "/unrestricted" {
      access_control = ["token"]
      required_permission = {
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
      required_permission = {
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
      required_permission = ""

      response {
        body = "a"
      }
    }

    endpoint "/wildcardAndMore" {
      allowed_methods = ["get", "*", "PuT", "brew"]
      required_permission = ""

      response {
        body = "a"
      }
    }

    endpoint "/blocked" {
      allowed_methods = []
      disable_access_control = ["token"]

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
    permissions_claim = "scope"
  }
}
