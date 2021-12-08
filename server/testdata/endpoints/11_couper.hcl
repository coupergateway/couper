server { # sequences
  endpoint "/simple" {
    request "resolve" {
      url = "${request.headers.origin}/"
      headers = {
        Accept = "application/json"
        X-Value = "my-value"
      }
    }

    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        x = backend_responses.resolve.headers.y-value
      }
      json_body = backend_responses.resolve.json_body
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

  endpoint "/simple-proxy-named" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        x = backend_responses.resolve.headers.y-value
      }
    }

    proxy "resolve" {
      url = "${request.headers.origin}/"
    }
  }

  endpoint "/complex-proxy" {
    request "resolve" { # use the reflected origin header to obtain the y-value
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

  endpoint "/parallel-complex-proxy" {
    request "standalone" { # parallel with both sequences
      url = "${backend_responses.resolve_first.headers.origin}/"
    }

    request "resolve" { # use the reflected origin header to obtain the y-value
      url = "${backend_responses.resolve_first.headers.origin}/"
    }

    request "resolve_first" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        origin = request.headers.origin
      }
    }

    request "resolve_gamma" { # use the reflected origin header to obtain the y-value
      url = "${backend_responses.resolve_gamma_first.headers.origin}/"
    }

    request "resolve_gamma_first" {
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

    response {
      headers = {
        x = backend_responses.default.headers.x
        y = backend_responses.resolve_gamma.headers.y-value
        z = backend_responses.standalone.headers.y-value
      }
    }
  }
}
