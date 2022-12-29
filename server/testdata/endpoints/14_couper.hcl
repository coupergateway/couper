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

    error_handler "endpoint" {
      response {
        status = 417
      }
    }

    error_handler "unexpected_status" {
      response {
        headers = {
          x = backend_responses.default.status
          y = backend_responses.default.json_body.JSON.list[0]
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
          y = backend_responses.default.json_body.JSON.list[0]
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

  endpoint "/sequence-break-unexpected_status" {
    request "resolve" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [418] # break
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.content-type
      }
      expected_status = [200]
    }
  }

  endpoint "/sequence-break-backend_timeout" {
    request "resolve" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      backend {
        timeout = "1ns" # break
      }
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.content-type
      }
      expected_status = [200]
    }
  }

  endpoint "/break-only-one-sequence" {
    request "resolve1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      expected_status = [418] # break
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve1.headers.content-type
      }
      expected_status = [200]
    }

    request "resolve2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      expected_status = [200]
    }

    proxy "refl" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve2.headers.content-type
      }
      expected_status = [200]
    }

    response {
      status = 200
    }
  }

  api {
    endpoint "/1.1" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/not-found"
        expected_status = [200]
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "unexpected_status" {
        response {
          headers = {
            handled-by = "unexpected_status"
          }
        }
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for unexpected_status and sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/1.2" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/not-found"
        expected_status = [200]
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for unexpected_status and sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/1.3" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/not-found"
        expected_status = [200]
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }
    }

    endpoint "/1.4" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/not-found"
        expected_status = [200]
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "endpoint" {  # super-type for unexpected_status and sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/1.5" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/not-found"
        expected_status = [200]
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }
    }

    endpoint "/2.1" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          timeout = "1ns"
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "backend_timeout" {
        response {
          headers = {
            handled-by = "backend_timeout"
          }
        }
      }

      error_handler "backend" {  # super-type for backend_timeout
        response {
          headers = {
            handled-by = "backend"
          }
        }
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/2.2" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          timeout = "1ns"
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "backend" {  # super-type for backend_timeout
        response {
          headers = {
            handled-by = "backend"
          }
        }
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/2.3" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          timeout = "1ns"
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/2.4" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          timeout = "1ns"
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/2.5" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          timeout = "1ns"
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }
    }

    endpoint "/3.1" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          openapi {
            file = "14_couper.yaml"
          }
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "backend_openapi_validation" {
        response {
          headers = {
            handled-by = "backend_openapi_validation"
          }
        }
      }

      error_handler "backend" {  # super-type for backend_openapi_validation
        response {
          headers = {
            handled-by = "backend"
          }
        }
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/3.2" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          openapi {
            file = "14_couper.yaml"
          }
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "backend" {  # super-type for backend_openapi_validation
        response {
          headers = {
            handled-by = "backend"
          }
        }
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/3.3" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          openapi {
            file = "14_couper.yaml"
          }
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "sequence" {
        response {
          headers = {
            handled-by = "sequence"
          }
        }
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/3.4" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          openapi {
            file = "14_couper.yaml"
          }
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }

      error_handler "endpoint" {  # super-type for sequence
        response {
          headers = {
            handled-by = "endpoint"
          }
        }
      }
    }

    endpoint "/3.5" {
      request "r1" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend {
          openapi {
            file = "14_couper.yaml"
          }
        }
      }

      request "r2" {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        json_body = backend_responses.r1.json_body
      }

      response {
        json_body = backend_responses.r2.json_body
      }
    }
  }
}
