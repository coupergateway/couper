server "backends" {
  api {
    endpoint "/anything" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR

          basic_auth = "${request.headers.x-user}:pass"
        }
      }
    }
  }
}
