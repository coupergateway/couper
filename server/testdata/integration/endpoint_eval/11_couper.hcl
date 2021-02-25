server "api" {
  api {
    endpoint "/**" {
      path = "/**"
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }
}
