server "defect" {
  error_file = "./../server_error.html"

  endpoint "/" {
    request {
      backend {
        origin = "http://:80/"
      }
    }
  }
}
