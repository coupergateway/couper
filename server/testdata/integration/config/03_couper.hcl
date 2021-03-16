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
            Authorization = req.headers.authorization # proxy blacklist
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
          Authorization = req.headers.authorization # proxy blacklist
        }
      }
    }
  }

  endpoint "/jwt" {
    disable_access_control = ["ba1"]
    access_control = ["JWTToken"]
    response {
      headers = {
        x-jwt-sub = req.ctx.JWTToken.sub
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
  }

  backend "test" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
    set_request_headers = {
      Authorization: req.headers.authorization
    }
  }
}

settings {
  no_proxy_from_env = true
}
