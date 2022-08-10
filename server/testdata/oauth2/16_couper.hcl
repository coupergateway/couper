server {
  api {
    endpoint "/" {
      proxy {
        url = "https://example.com/"

        backend {
          oauth2 {
            token_endpoint = "{{.asOrigin}}/options"
            grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
            assertion      = request.method # easier for test purpose, should of course be a signed JWT
          }
        }
      }
    }
  }
}
