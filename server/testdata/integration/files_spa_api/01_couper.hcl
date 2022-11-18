server "spa" {
  error_file = "./../server_error.html"
  files {
    document_root = "./"
  }
  spa {
    bootstrap_file = "01_app.html"
    paths = ["/**"]
    bootstrap_data = {
      default = default(env.NOT_THERE, "true")
    }
  }
  api {
    error_file = "./../api_error.json"
    endpoint "/api" {
      proxy {
        backend = "anything"
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
