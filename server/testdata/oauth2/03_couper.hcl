server "oauth2-options" {
  error_file = "./../integration/server_error.html"

  api {
    error_file = "./../integration/api_error.json"

    endpoint "/" {
      proxy {
        backend {
          url = "https://example.com/"

          oauth2 {
            token_endpoint = "${request.headers.x-as-origin}/options"
            retries        = 0
            client_id      = "user"
            client_secret  = "pass"
            grant_type     = "client_credentials"
          }
        }
      }
    }
  }
}
