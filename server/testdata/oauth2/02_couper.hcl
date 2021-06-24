server "oauth2-options" {
  error_file = "./testdata/integration/server_error.html"

  api {
    error_file = "./testdata/integration/api_error.json"

    endpoint "/" {
      proxy {
        backend {
          url = "https://example.com/"

          oauth2 {
            token_endpoint = "{{.asOrigin}}/options"
            retries        = 0
            client_id      = "user"
            client_secret  = "pass"
            grant_type     = "client_credentials"
            token_endpoint_auth_method = "client_secret_basic"
          }
        }
      }
    }
  }
}
