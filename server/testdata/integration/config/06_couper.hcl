server "cors" {
  access_control = ["ba"]

  files {
    document_root = "./"
    cors {
      allowed_origins = "a.com"
    }
  }

  spa {
    paths = ["/spa"]
    bootstrap_file = "06_couper.hcl"
    cors {
      allowed_origins = "b.com"
    }
  }

  api {
    base_path = "/api"
    cors {
      allowed_origins = "c.com"
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
