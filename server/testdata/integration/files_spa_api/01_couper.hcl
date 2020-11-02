server "spa" {
  files {
    document_root = "./"
  }
  spa {
    bootstrap_file = "app.html"
    paths = ["/**"]
  }
  api {
    endpoint "/api" {
      path = "/"
      backend {
        origin = "https://httpbin.org"
      }
    }
  }
}
