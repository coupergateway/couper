server "api" {
  api {
    base_path = "/v1"

    endpoint "/" {
      backend {
        path = "/anything"
        origin = "http://anyserver/"
      }
    }
  }
}
