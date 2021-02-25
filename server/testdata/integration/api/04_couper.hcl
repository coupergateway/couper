server "ws" {
  api {
    endpoint "/upgrade" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          # /ws path is a echo websocket upgrade handler at our test-backend
          path = "/ws"
        }
      }
    }
  }
}

settings {
  no_proxy_from_env = true
}
