server "free-endpoint" {
  endpoint "/" {
    proxy {
      backend {
        set_query_params = {
          test = "pest"
        }

        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }
}
