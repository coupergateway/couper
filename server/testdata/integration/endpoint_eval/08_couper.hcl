server "protected" {
  error_file = "./../server_error.html"

  api {
    error_file = "./../api_error.json"

    backend "anything" { # overrides definitions: backend "anything"
      path = "/set/by/api/unset/by/endpoint"
      openapi {
        file = "also-not-there.yml"
      }
    }

    endpoint "/{origin}" {
      path = "/set/by/endpoint/unset/by/backend"
      backend "anything" {
        path = "/anything"
        origin = "http://${req.path_params.origin}"
        hostname = req.path_params.hostname
        set_response_headers = {
          x-origin = req.path_params.origin
        }
        openapi {
          file = "08_schema.yaml"
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
    openapi {
      file = "not-there.yml"
    }
  }
}
