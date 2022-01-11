server {
  endpoint "/**" {

    request "sidekick" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}"
    }

    request {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
      }
    }
  }
}
