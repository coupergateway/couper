server {
  endpoint "/" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "xxx"
      }
    }
  }
}
