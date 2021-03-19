server "api" {
  error_file = "./../integration/server_error.html"

  api {
    base_path = "/v1"

    error_file = "./../integration/api_error.json"

    endpoint "/" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/proxy"
        backend = "proxy"
      }
      request "request" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/request"
        backend = "request"
      }
      response {
        status = beresp.status + 1
		# 404 + 404
        body = beresps.request.status + beresps.default.status
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "proxy" {
    path = "/override/me"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
  backend "request" {
    path = "/override/me"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
