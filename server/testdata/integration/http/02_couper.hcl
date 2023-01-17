server "second" {
  endpoint "/" {
    proxy {
      backend = "b1"
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
