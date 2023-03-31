server "second" {
  endpoint "/" {
    proxy {
      backend = "b1"
    }

    request "REQ" {
      backend = "b2"
    }
  }
  endpoint "/seq" {
    proxy {
      backend = "b1"
      set_request_headers = {
        x-req = backend_responses.REQ.status
      }
    }

    request "REQ" {
      backend = "b2"
    }
  }
}

settings {
  server_timing_header = true
}

definitions {
  backend "b1" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/anything"
  }

  backend "b2" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/small"
  }
}
