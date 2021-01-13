server "free-endpoint" {
  endpoint "/" {
    backend {
      set_query_params = {
        test = "pest"
      }

      origin = env.COUPER_TEST_BACKEND_ADDR
    }
  }
}
