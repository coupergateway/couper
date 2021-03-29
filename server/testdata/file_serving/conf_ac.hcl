server "TestFileServing" {
  hosts = ["protect.me"]

  access_control = ["ba"]

  error_file = "./error.html"

  files {
    document_root = "./htdocs"
  }

  spa {
    base_path = "/app"
    bootstrap_file = "./htdocs/spa.html"
    paths = ["/**"]
  }
}

definitions {
  basic_auth "ba" {
    password = "hans"
  }
}
