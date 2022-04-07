server {
  endpoint "/" {
    proxy {
      backend {
        origin = "https://httpbin.org"
        path = "/anything"
      }
    }

    request "req" {
      backend {
        origin = "https://httpbin.org"
        path = "/anything"
      }
    }
  }

  endpoint "/be" {
    proxy {
      backend = "be"
    }
  }

  endpoint "/secure" {
    proxy {
      backend "be" {
        oauth2 {
          backend {
            origin = "https://httpbin.org"
            path = "/anything"

            oauth2 {
              backend {
                origin = "https://httpbin.org"
                path = "/anything"
              }

              client_id = "foo"
              client_secret = "5eCr3t"
              grant_type = "client_credentials"
            }
          }

          client_id = "foo"
          client_secret = "5eCr3t"
          grant_type = "client_credentials"
        }
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "https://httpbin.org"
    path = "/anything"
  }

  jwt "jwt" {
    backend {
      origin = "https://httpbin.org"
      path = "/anything"
    }

    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
  }

  beta_oauth2 "oauth2" {
    backend {
      origin = "https://httpbin.org"
      path = "/anything"
    }

    authorization_endpoint = "https://authorization.server/oauth/authorize"
    client_id = "foo"
    client_secret = "5eCr3t"
    grant_type = "authorization_code"
    redirect_uri = "http://localhost:8085/oidc/callback"
    token_endpoint = "https://authorization.server/oauth/token"
    verifier_method = "ccm_s256"
    verifier_value = "not_used_here"
  }

  beta_oidc "oidc" {
    backend {
      origin = "https://httpbin.org"
      path = "/anything"
    }

    client_id = "foo"
    client_secret = "etbinbp4in"
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    redirect_uri = "http://localhost:8080/cb"
    verifier_value = request.cookies.nnc
  }
}
