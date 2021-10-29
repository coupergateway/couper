server "cors" {
  access_control = ["ba"]

  files {
    document_root = "./"
    cors {
      allowed_origins = "a.com"
      allow_credentials = true
    }
  }

  spa {
    paths = ["/spa"]
    bootstrap_file = "06_couper.hcl"
    cors {
      allowed_origins = "b.com"
      allow_credentials = true
    }
  }

  api {
    base_path = "/api"
    cors {
      allowed_origins = "c.com"
      allow_credentials = true
    }
    endpoint "/" {
      response {}
    }
  }
}
definitions {
  basic_auth "ba" {
    user = "foo"
    password = "asdf"
  }
}
