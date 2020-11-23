server "api" {
  error_file = "./../server_error.html"

  api {
    base_path = "/v1"

    error_file = "./../api_error.json"

    endpoint "/" {
      backend = "anything"
    }

    endpoint "/connect-error" {
      backend {
        origin = "http://1.2.3.4"
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = "http://anyserver/"
  }
}
