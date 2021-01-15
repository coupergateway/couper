server "api" {
  api {
    endpoint "/**" {
      path = "/**"
      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }
}
