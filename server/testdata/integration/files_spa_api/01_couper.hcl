server "spa" {
  error_file = "./../server_error.html"
  files {
    document_root = "./"
  }
  spa {
    bootstrap_file = "app.html"
    paths = ["/**"]
  }
  api {
    error_file = "./../api_error.json"
    endpoint "/api" {
      path = "/"
      backend {
        origin = "https://httpbin.org"
      }
    }
  }
}
