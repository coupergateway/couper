server {
  api {
    endpoint "/" {
      proxy {
        url = "https://example.com/"

        backend {
          oauth2 {
            token_endpoint = "{{.asOrigin}}/token"
            grant_type     = "urn:ietf:params:oauth:grant-type:jwt-bearer"
            jwt_signing_profile {
              signature_algorithm = "HS256"
              key = "asdf"
              ttl = "10s"
              claims = {
                iss = "foo@example.com"
                scope = "sc1 sc2"
                aud = "https://authz.server/token"
                iat = unixtime()
              }
            }
          }
        }
      }
    }
  }
}
