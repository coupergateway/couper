server { # sequences
  endpoint "/simple" {
    request "resolve" {
      url = "${request.headers.origin}/"
    }

    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        x = backend_responses.resolve.headers.y-value
      }
    }
  }

  endpoint "/simple-proxy" {
    request "resolve" {
      url = "${request.headers.origin}/"
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.y-value
      }
    }
  }

  endpoint "/complex-proxy" {
    request "resolve" {
      url = "${backend_responses.resolve_first.headers.origin}/"
    }

    request "resolve_first" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        origin = request.headers.origin
      }
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.y-value
      }
    }
  }
}
