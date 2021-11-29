server "logs" {
  files {
    document_root = "./"
    custom_log_fields = {
      files = request.method
    }
  }

  spa {
    bootstrap_file = "./file.html"
    paths = ["/spa/**"]
    custom_log_fields = {
      spa = request.method
    }
  }

  custom_log_fields = {
    server = backend_responses.default.headers.server
  }

  endpoint "/secure" {
    access_control = ["BA"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt-valid" {
    access_control = ["JWT"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt" {
    access_control = ["JWT"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt-wildcard" {
    access_control = ["JWT-WILDCARD"]

    proxy {
      backend = "BE"
    }
  }

  api {
    custom_log_fields = {
      api = backend_responses.default.headers.server
    }

    endpoint "/" {
      custom_log_fields = {
        endpoint = backend_responses.default.headers.server
      }

      proxy {
        backend "BE" {
          custom_log_fields = {
            bool   = true
            int    = 123
            float  = 1.23
            string = backend_responses.default.headers.server
            req    = request.method

            array = [
              1,
              backend_responses.default.headers.server,
              [
                2,
                backend_responses.default.headers.server
              ],
              {
                x = "X"
              }
            ]

            object = {
              a = "A"
              b = "B"
              c = 123
            }
          }
        }
      }
    }

    endpoint "/backend" {
      proxy {
        backend = "BE"
      }
    }
  }
}

definitions {
  backend "BE" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path   = "/anything"

    custom_log_fields = {
      backend = backend_responses.default.headers.server
    }
  }

  basic_auth "BA" {
    password = "secret"

    error_handler "basic_auth" {
      custom_log_fields = {
        error_handler = request.method
      }
    }
  }

  jwt "JWT" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"

    custom_log_fields = {
      jwt_regular = request.method
    }

    error_handler "jwt" {
      custom_log_fields = {
        jwt_error = request.method
      }
    }
  }

  jwt "JWT-WILDCARD" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"

    custom_log_fields = {
      jwt_regular = request.method
    }

    error_handler {
      custom_log_fields = {
        jwt_error_wildcard = request.method
      }
    }
  }
}
