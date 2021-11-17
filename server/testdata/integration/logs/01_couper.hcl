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
    server = backend_responses.default.json_body.Method
  }

  endpoint "/secure" {
    access_control = ["BA"]

    proxy {
      backend = "BE"
    }
  }

  api {
    custom_log_fields = {
      api = backend_responses.default.json_body.Method
    }

    endpoint "/" {
      custom_log_fields = {
        endpoint = backend_responses.default.json_body.Method
      }

      proxy {
        backend "BE" {
          custom_log_fields = {
            bool   = true
            int    = 123
            float  = 1.23
            string = backend_responses.default.json_body.Method

            array = [
              1,
              backend_responses.default.json_body.Method,
              [
                2,
                backend_responses.default.json_body.Method
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
      backend = backend_responses.default.json_body.Method
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
}
