server "api" {
  error_file = "./../integration/server_error.html"

  api {
    base_path = "/v1"

    error_file = "./../integration/api_error.json"

    endpoint "/" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/proxy"
        backend = "proxy"
        set_request_headers = {
          x-inline = "test"
        }
      }

      proxy "p2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/proxy"
        backend = "proxy"
        set_request_headers = {
          x-inline = "test"
        }
      }

      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/request"
        backend = "request"
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/request"
        backend = "request"
      }

      set_request_headers = {
        x-ep-inline = "test"
      }

      response {
        status = "${backend_responses.default.status}" # string type test
        # 404 + 404 + 404 + 404
        body = backend_responses.r1.status + backend_responses.default.status + backend_responses.r2.status + backend_responses.p2.status
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "proxy" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
    set_request_headers = {
      x-data = "proxy-test"
    }
  }
  backend "request" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
    set_request_headers = {
      x-data = "request-test"
    }
  }
}
