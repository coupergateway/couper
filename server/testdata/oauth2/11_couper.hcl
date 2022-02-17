server "oauth-client" {
  api {
    endpoint "/rs1" {
      proxy {
        backend = "rs1"
      }
    }
    endpoint "/rs2" {
      proxy {
        backend = "rs2"
      }
    }
  }
}

definitions {
  # backends for resource server
  # with referenced backend
  backend "rs1" {
    origin = "{{.rsOrigin}}"
    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      backend = "token"
      client_id = "foo"
      client_secret = "etbinbp4in"
      grant_type = "client_credentials"
    }
  }
  # with inline backend
  backend "rs2" {
    origin = "{{.rsOrigin}}"
    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      backend {
        origin = "{{.asOrigin}}"
        add_request_headers = {
          x-sub = "myself"
        }
      }
      client_id = "foo"
      client_secret = "etbinbp4in"
      grant_type = "client_credentials"
    }
  }

  backend "token" {
    origin = "{{.asOrigin}}"
    add_request_headers = {
      x-sub = "myself"
    }
  }

}

settings {
  no_proxy_from_env = true
}
