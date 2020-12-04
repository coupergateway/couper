server "api" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    backend "anything" { # overrides definitions: backend "anything"
      path = "/set/by/api/unset/by/endpoint"
    }

    endpoint "/{path}/{hostname}/{origin}" {
      path = "/set/by/endpoint/unset/by/backend"
      backend "anything" {
        path = "/anything"
        origin = "http://${req.path_params.origin}"
        hostname = req.path_params.hostname
        response_headers = {
          x-origin = req.path_params.origin
        }
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/not-found-anything"
    origin = "http://anyserver/"
  }
}
