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
