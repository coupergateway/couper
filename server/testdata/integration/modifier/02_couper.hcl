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
}