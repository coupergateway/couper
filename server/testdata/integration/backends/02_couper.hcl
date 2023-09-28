server {
  hosts = ["*:8080"]
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
          x-from-response-json-body = backend_response.json_body.URL
          x-from-responses-json-body = backend_responses.default.json_body.URL
        }
        custom_log_fields = {
          x-from-request-header = backend_request.headers.x-foo
          x-from-requests-header = backend_requests.default.headers.x-foo
          x-from-request-json-body = backend_request.json_body.a
          x-from-requests-json-body = backend_requests.default.json_body.a
          x-from-response-header = backend_response.headers.content-type
          x-from-responses-header = backend_responses.default.headers.content-type
          x-from-response-json-body = backend_response.json_body.URL
          x-from-responses-json-body = backend_responses.default.json_body.URL
        }

        oauth2 {
          grant_type = "client_credentials"
          token_endpoint = "http://localhost:8081/token"
          client_id = "qpeb"
          client_secret = "ben"
          backend {
            set_response_headers = {
              # use a response header field that is actually logged
              location = "${backend_request.headers.authorization}|${backend_request.form_body.grant_type[0]}|${backend_response.headers.x-pires-in}|${backend_response.json_body.access_token}"
            }
            custom_log_fields = {
              x-from-request-header = backend_request.headers.authorization
              x-from-request-body = backend_request.body
              x-from-request-form-body = backend_request.form_body.grant_type[0]
              x-from-response-header = backend_response.headers.x-pires-in
              x-from-response-body = backend_response.body
              x-from-response-json-body = backend_response.json_body.access_token
            }
          }
        }
      }
    }
  }

  endpoint "/request2" {
    request "r" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      json_body = {
        b = 2
      }
      backend {
        custom_log_fields = {
          x-from-requests-body = backend_requests.r.body
          x-from-requests-json-body = backend_requests.r.json_body.b
        }
      }
    }
    response {
      headers = {
        x-from-requests-body = backend_requests.r.body
        x-from-requests-json-body = backend_requests.r.json_body.b
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

server "as" {
  hosts = ["*:8081"]
  api {
    endpoint "/token" {
      response {
        headers = {
          x-pires-in = "60s"
        }
        json_body = {
          access_token = "the_access_token"
          expires_in = 60
        }
      }
    }
  }
}
