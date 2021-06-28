server "api" {
  error_file = "./testdata/integration/server_error.html"

  api {
    error_file = "./testdata/integration/api_error.json"

    endpoint "/" {
      proxy {
        backend {
          origin = "{{.rsOrigin}}"
          path   = "/resource"

          oauth2 {
            token_endpoint = "{{.asOrigin}}/oauth2"
            client_id      = "user"
            client_secret  = "pass word"
            grant_type     = "client_credentials"
            retries        = 0
          }
        }
      }
    }

    endpoint "/2nd" {
      proxy {
        backend {
          origin = "{{.rsOrigin}}"
          path   = "/resource"

          oauth2 {
            client_id      = "user"
            client_secret  = "pass word"
            grant_type     = "client_credentials"
            retries        = 0
            backend {
              origin = "{{.asOrigin}}"
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
