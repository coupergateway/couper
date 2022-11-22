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
      headers = {
        Accept = "application/json"
        X-Value = "my-proxy-value"
      }
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      set_request_headers = {
        x = backend_responses.resolve.headers.y-value
        y = backend_responses.resolve.body
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

  endpoint "/parallel-complex-nested" {
    request "last" { # seq waiting for the other sequences in parallel
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      headers = {
        a = backend_responses.resolve_gamma.headers.y-value
        b = backend_responses.resolve.headers.y-value
      }
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
        a = backend_responses.last.headers.a
        b = backend_responses.last.headers.b
        x = backend_responses.default.headers.x
        y = backend_responses.resolve_gamma.headers.y-value
      }
    }
  }

  endpoint "/multiple-request-uses" {
    request "r1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
    }
    request "r2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = backend_responses.r1.json_body
    }
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.r1.json_body
        ,
        backend_responses.r2.json_body
      ]
    }
  }

  endpoint "/multiple-proxy-uses" {
    proxy "p1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
    }
    request "r2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = backend_responses.p1.json_body
    }
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.p1.json_body
        ,
        backend_responses.r2.json_body
      ]
    }
  }

  endpoint "/multiple-sequence-uses" {
    request "r1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
    }
    request "r2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = backend_responses.r1.json_body
    }
    request "r3" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = backend_responses.r2.json_body
    }
    request "r4" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = backend_responses.r2.json_body
    }
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.r3.json_body
        ,
        backend_responses.r4.json_body
      ]
    }
  }

  endpoint "/multiple-parallel-uses" {
    request "r1" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
    }
    request "r2" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
    }
    request "r3" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.r1.json_body
        ,
        backend_responses.r2.json_body
      ]
    }
    request "r4" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.r1.json_body
        ,
        backend_responses.r2.json_body
      ]
    }
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflect"
      json_body = [
        backend_responses.r3.json_body
        ,
        backend_responses.r4.json_body
      ]
    }
  }
}
