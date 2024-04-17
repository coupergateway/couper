server {
  endpoint "/buffer" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/json"
    }
  }

  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/reflectDelay"
    }
  }
}
