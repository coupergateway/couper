server "oauth2-options" {
  api {
    endpoint "/" {
      proxy {
        url = "https://example.com/"

        backend {
          oauth2 {
            token_endpoint = "{{.asOrigin}}/options"
            client_id      = "my_client"
            client_secret  = "my_client_secret"
            grant_type     = "password"
            username       = "user"
            password       = "pass"
            scope          = "scope1 scope2"
          }
        }
      }
    }
  }
}
