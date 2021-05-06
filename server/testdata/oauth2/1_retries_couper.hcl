server "api" {
  error_file = "./../integration/server_error.html"

  api {
    error_file = "./../integration/api_error.json"

    endpoint "/" {
      proxy {
        backend {
          origin = "${request.headers.x-origin}"
          path   = "/resource"

          oauth2 {
            token_endpoint = "${request.headers.x-token-endpoint}/oauth2"
            client_id      = "user"
            client_secret  = "pass"
            grant_type     = "client_credentials"
            token_endpoint_auth_method = "client_secret_basic"
          }
        }
      }
    }

    endpoint "/2nd" {
      proxy {
        backend {
          origin = "${request.headers.x-origin}"
          path   = "/resource"

          oauth2 {
            client_id      = "user"
            client_secret  = "pass"
            grant_type     = "client_credentials"
            token_endpoint_auth_method = "client_secret_basic"
            backend {
              origin = "${request.headers.x-token-endpoint}"
              path = "/oauth2"
            }
          }
        }
      }
    }
  }
}

settings {
  no_proxy_from_env = true
}
