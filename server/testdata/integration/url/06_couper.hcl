server "url" {
  api {
    endpoint "/" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything?x=y"
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          set_query_params = {
            a = "A"
          }
        }
      }
    }
  }
}
