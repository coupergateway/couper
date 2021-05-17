server "api" {
  api {
    endpoint "/**" {
      path = "/new/path/**"
      proxy {
        backend {
          origin = "${env.COUPER_TEST_BACKEND_ADDR}/?q#f"
          path = "/path#?xxx"
        }
      }
    }
  }
}
