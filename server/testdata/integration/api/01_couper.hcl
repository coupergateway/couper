server "api" {
  error_file = "./../server_error.html"
  api {
    base_path = "/v1"

    error_file = "./../api_error.json"

    endpoint "/" {
      backend {
        path = "/anything"
        origin = "http://anyserver/"
      }
    }
  }
}
