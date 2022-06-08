server "api" {
  api {
    endpoint "/**" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }
}
