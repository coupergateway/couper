server { # sequences
  endpoint "/" {
    request "resolve" { # use the reflected origin header to obtain the y-value
      backend {
        origin = "${backend_responses.resolve_first.headers.origin}"
        hostname = "test.local"
        path = "/"
        timeout = "950ms"
      }
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
