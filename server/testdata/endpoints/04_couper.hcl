server "api" {
  error_file = "./../integration/server_error.html"

  api {
    error_file = "./../integration/api_error.json"

    endpoint "/" {
      proxy {
        backend {
          origin = "http://example.com"
		  path   = "/resource"

		  oauth2 {
            token_endpoint = "${req.headers.token-endpoint}"
            client_id      = "user"
            client_secret  = "pass"
            grant_type     = "client_credentials"
          }
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
