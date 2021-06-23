server "set-response-status" {
  endpoint "/204" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      backend {
        set_response_status = 204
      }
    }
  }
  endpoint "/201" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      backend {
        set_response_status = 201
      }
    }
  }
  endpoint "/600" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      backend {
        set_response_status = 600
      }
    }
  }

  endpoint "/teapot" {
    access_control = ["ba"]
    response {}
  }

  endpoint "/no-content" {
    response {
      status = 500
    }
    set_response_status = 204
  }

  endpoint "/happy-path-only" {
    proxy {
      url = "couper://some.host/"
    }
    set_response_status = 418
  }

  endpoint "/inception" {
    access_control = ["layer2"]
    response {}
  }
}

definitions {
  basic_auth "ba" {
    user = "hans"
    password = "peter"
    error_handler {
      set_response_status = 418
    }
  }

  basic_auth "layer2" {
    password = "sauerkraut"
    error_handler {
      request {
        url = "couper://some.host/"
      }
      set_response_status = 418
    }
  }
}

