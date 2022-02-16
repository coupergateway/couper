server {
  endpoint "/" {
    proxy "default" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      backend {
        custom_log_fields = {
          default-res = backend_response.headers.content-type
          default-req = backend_request.headers.cookie
          default-ua  = backend_request.headers.user-agent
        }

        set_response_headers = {
          test-header = backend_response.headers.content-type
        }
      }
    }

    request "request" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/small"

      backend { 
        custom_log_fields = {
          request-res = backend_response.headers.content-type
          request-req = backend_request.headers.cookie
          request-ua  = backend_request.headers.user-agent
        }
      }
    }

    request "r1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/small"
      backend = "BE"
    }

    request "r2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      backend = "BE"
    }
  }

  endpoint "/request" {
    request "default" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      headers = {
        x-foo = "bar"
      }
      json_body = {
        a = 1
      }

      backend {
        set_response_headers = {
          x-from-request-header = backend_request.headers.x-foo
          x-from-requests-header = backend_requests.default.headers.x-foo 
          x-from-request-json-body = backend_request.json_body.a
          x-from-requests-json-body = backend_requests.default.json_body.a
          x-from-response-header = backend_response.headers.content-type
          x-from-responses-header = backend_responses.default.headers.content-type
          x-from-response-json-body = backend_response.json_body.Url
          x-from-responses-json-body = backend_responses.default.json_body.Url
        }
        custom_log_fields = {
          x-from-request-header = backend_request.headers.x-foo
          x-from-requests-header = backend_requests.default.headers.x-foo
          x-from-request-json-body = backend_request.json_body.a
          x-from-requests-json-body = backend_requests.default.json_body.a
          x-from-response-header = backend_response.headers.content-type
          x-from-responses-header = backend_responses.default.headers.content-type
          x-from-response-json-body = backend_response.json_body.Url
          x-from-responses-json-body = backend_responses.default.json_body.Url
        }
      }
    }
  }
}

definitions {
  backend "BE" {
    custom_log_fields = {
      definitions-res = backend_response.headers.content-type
      definitions-req = backend_request.headers.cookie
      definitions-ua  = backend_request.headers.user-agent
    }
  }
}
