server "api" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    endpoint "/{path}/{hostname}/{origin}" {
      path = "/set/by/endpoint/unset/by/backend"
      proxy {
        backend "anything" {
          path = "/anything"
          origin = "http://${req.path_params.origin}"
          hostname = req.path_params.hostname
          set_response_headers = {
            x-origin = req.path_params.origin
          }
        }
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/not-found-anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
