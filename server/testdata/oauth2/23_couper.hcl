server {
  api {
    endpoint "/" {
      proxy {
        url = "https://example.com/"

        backend {
          oauth2 {
            token_endpoint = "{{.asOrigin}}/token"
            grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
            assertion      = request.headers.x-assertion
          }
        }
      }
    }
  }
}
