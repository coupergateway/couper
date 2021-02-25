server "api" {
  error_file = "./../server_error.html"

  api {
    base_path = "/v1"

    error_file = "./../api_error.json"

    endpoint "/" {
      proxy {
        backend = "anything"
      }
    }

    endpoint "/proxy" {
      proxy {
        backend {
          origin = "http://example.com"
        }
      }
    }

    endpoint "/connect-error" {
      proxy {
        backend {
          connect_timeout = "2s"
          origin = "http://1.2.3.4"
        }
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
