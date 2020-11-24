server "api" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    endpoint "/{path}/{hostname}/{origin}" {
      path = "/unset/by/backend"
      backend {
        path = "/anything"
        origin = "http://${req.path_param.origin}"
        hostname = req.path_param.hostname
        response_headers = {
          x-origin = req.path_param.origin
        }
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
