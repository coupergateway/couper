server "logs" {
  files {
    document_root = "./"
    log_fields = {
      files = request.method
    }
  }

  spa {
    bootstrap_file = "./file.html"
    paths = ["/spa/**"]
    log_fields = {
      spa = request.method
    }
  }

  log_fields = {
    server = backend_responses.default.json_body.Method
  }

  endpoint "/secure" {
    access_control = ["BA"]

    proxy {
      backend = "BE"
    }
  }

  api {
    log_fields = {
      api = backend_responses.default.json_body.Method
    }

    endpoint "/" {
      log_fields = {
        endpoint = backend_responses.default.json_body.Method
      }

      proxy {
        backend "BE" {
          log_fields = {
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
  }
}

definitions {
  backend "BE" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path   = "/anything"
  }

  basic_auth "BA" {
    password = "secret"

    error_handler "basic_auth" {
      log_fields = {
        error_handler = request.method
      }
    }
  }
}
