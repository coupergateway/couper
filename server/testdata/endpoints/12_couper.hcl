server { # sequences
  endpoint "/" {
    request "resolve" { # use the reflected origin header to obtain the y-value
      url = "${backend_responses.resolve_first.headers.origin}/" # this backend will trigger a client cancel
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
