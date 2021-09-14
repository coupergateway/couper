server "acs" {
  access_control = ["ba1"]
  error_file = "../api_error.json"
  api {
    base_path = "/v1"
    disable_access_control = ["ba1"]
    endpoint "/**" {
      # access_control = ["ba1"] # not possible atm TODO: spec
      proxy {
        backend "test" {
          set_request_headers = {
            auth = ["ba1"]
          }
        }
      }
    }
  }

  api {
    base_path = "/v2"
    access_control = ["ba2"]
    endpoint "/**" {
      proxy {
        backend "test" {
          set_request_headers = {
            auth = ["ba1", "ba2"]
            Authorization = request.headers.authorization # proxy blacklist
          }
        }
      }
    }
  }

  api {
    base_path = "/v3"
    access_control = ["ba2"]
    endpoint "/**" {
      access_control = ["ba3"]
      disable_access_control = ["ba1", "ba2", "ba3"]
      proxy {
        backend = "test"
      }
    }
  }

  api {
    base_path = "/v4"
    access_control = ["ba2"]
    endpoint "/**" {
      error_file = "../server_error.html" # error_file in endpoint
      proxy {
        backend = "test"
      }
    }
  }

  api {
    base_path = "/v5"
    access_control = ["ba2"]
    endpoint "/exists" {
      error_file = "../server_error.html" # error_file in endpoint
      proxy {
        backend = "test"
      }
    }
  }

  endpoint "/status" {
    disable_access_control = ["ba1"]
    proxy {
      backend = "test"
    }
  }

  endpoint "/superadmin" {
    access_control = ["ba4"]
    proxy {
      backend "test" {
        set_request_headers = {
          auth = ["ba1", "ba4"]
          Authorization = request.headers.authorization # proxy blacklist
        }
      }
    }
  }

  endpoint "/jwt" {
    disable_access_control = ["ba1"]
    access_control = ["JWTToken"]
    response {
      headers = {
        x-jwt-sub = request.context.JWTToken.sub
        x-scopes = json_encode(request.context.scopes)
      }
    }
  }

  endpoint "/jwt/rsa" {
    disable_access_control = ["ba1"]
    access_control = ["RSAToken"]
    response {
      headers = {
        x-jwt-sub = request.context.RSAToken.sub
      }
    }
  }

  endpoint "/jwks/rsa" {
    disable_access_control = ["ba1"]
    access_control = ["JWKS"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKS.sub
      }
    }
  }

  endpoint "/jwks/rsa/remote" {
    disable_access_control = ["ba1"]
    access_control = ["JWKSRemote"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKSRemote.sub
      }
    }
  }
}

definitions {
  basic_auth "ba1" {
    password = "asdf"
  }
  basic_auth "ba2" {
    password = "asdf"
  }
  basic_auth "ba3" {
    password = "asdf"
  }
  basic_auth "ba4" {
    password = "asdf"
  }
  jwt "JWTToken" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    beta_scope_claim = "scope"
  }
  jwt "RSAToken" {
    header = "Authorization"
    signature_algorithm = "RS256"
    key_file = "../files/certificate.pem"
  }
  jwt "JWKS" {
    header = "Authorization"
    jwks_uri = "file:../files/jwks.json"
  }
  jwt "JWKSRemote" {
    header = "Authorization"
    jwks_uri = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
  }
  backend "test" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
    set_request_headers = {
      Authorization: request.headers.authorization
    }
  }
}

settings {
  no_proxy_from_env = true
}
