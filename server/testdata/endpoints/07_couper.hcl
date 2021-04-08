server "protected" {
  api {
    base_path      = "/v1"
    error_file     = "./../integration/api_error.json"
    access_control = ["BA"]

    endpoint "/{path}" {
      proxy {
        backend {
          path   = request.path_params.path
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }
}

definitions {
  basic_auth "BA" {
    password = "secret"
  }
}
