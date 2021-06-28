server "couper" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    endpoint "/" {
      proxy {
        backend = "anything"
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/error"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
