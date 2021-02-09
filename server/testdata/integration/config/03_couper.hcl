server "acs" {
  access_control = ["ba1"]
  backend = "test"
  error_file = "../api_error.json"
  api {
    base_path = "/v1"
    disable_access_control = ["ba1"]
    endpoint "/**" {
      # access_control = ["ba1"] # not possible atm TODO: spec
      backend "test" {
        set_request_headers = {
          auth = ["ba1"]
        }
      }
    }
  }

  api {
    base_path = "/v2"
    access_control = ["ba2"]
    endpoint "/**" {
      backend "test" {
        set_request_headers = {
          auth = ["ba1", "ba2"]
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
    }
  }

  endpoint "/status" {
    disable_access_control = ["ba1"]
  }

  endpoint "/superadmin" {
    access_control = ["ba4"]
    backend "test" {
      set_request_headers = {
        auth = ["ba1", "ba4"]
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
