server { # error_handler
  endpoint "/ok" {
    request "resolve" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [200, 204]
    }

    custom_log_fields = {
      beresp_res = backend_responses.resolve
      beresp_def = backend_responses.default
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.content-type
      }
    }

    error_handler "unexpected_status" {
      response {
        status = 418
      }
    }
  }

  endpoint "/not-ok" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [418]
    }

    error_handler "unexpected_status" {
      response {
        headers = {
          x = backend_responses.default.status
          y = backend_responses.default.json_body.Json.list[0]
        }
        status = 418
      }
    }
  }

  endpoint "/not-ok-endpoint" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [418]
    }

    error_handler "endpoint" {
      response {
        headers = {
          x = backend_responses.default.status
          y = backend_responses.default.json_body.Json.list[0]
        }
        status = 418
      }
    }
  }

  endpoint "/not-ok-sequence" {
    request "resolve" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [200, 204]
    }

    custom_log_fields = {
      beresp_res = backend_responses.resolve
      beresp_def = backend_responses.default
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.content-type
      }
      expected_status = [418]
    }

    error_handler "unexpected_status" {
      response {
        headers = {
          x = backend_responses.default.headers.x
        }
        status = 418
      }
    }
  }
}
