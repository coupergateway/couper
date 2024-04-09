server "cors" {
  access_control = ["ba"]

  files {
    document_root = "./"
    cors {
      allowed_origins = "a.com"
      allow_credentials = true
      max_age = "200s"
    }
  }

  spa {
    paths = ["/spa"]
    bootstrap_file = "06_couper.hcl"
    cors {
      allowed_origins = "b.com"
      allow_credentials = true
      max_age = "200s"
    }
  }

  api {
    base_path = "/api"
    cors {
      allowed_origins = "c.com"
      allow_credentials = true
      max_age = "200s"
    }
    endpoint "/" {
      response {
        headers = {
          access-control-allow-origin = "foo"
          access-control-allow-credentials = "bar"
          access-control-allow-methods = "BREW"
          access-control-allow-headers = "Auth"
          access-control-max-age = 300
        }
      }
    }
  }
}
definitions {
  basic_auth "ba" {
    user = "foo"
    password = "asdf"
  }
}
