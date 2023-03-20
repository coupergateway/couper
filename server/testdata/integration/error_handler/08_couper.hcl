server {
  api {
    endpoint "/anything" {
      request "r" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/"
        backend {
          timeout = "1ns"
        }
        body = "foo"
      }

      proxy {
        backend {
          origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        }
      }
    }

    error_handler "backend_timeout" {
      response {
        status = 418
        json_body = {
          resp_json_status = backend_responses.default.json_body.ResponseStatus
        }
      }
    }
  }
}
