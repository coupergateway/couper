server "ws" {
  api {
    endpoint "/upgrade/**" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          # /ws path is a echo websocket upgrade handler at our test-backend
          path = "/**"
        }

        websockets {
          set_request_headers = {
            Echo = "ECHO"
          }

          set_response_headers = {
            Abc = "123"
            X-Upgrade-Body = request.body
            X-Upgrade-Resp-Body = backend_responses.default.body # should not be set due to upgrade
          }
        }

        # affects both cases: upgrade and non 101
        set_response_headers = {
          X-Body = request.body
          X-Resp-Body = backend_responses.default.body
        }
      }
    }
  }
}

settings {
  no_proxy_from_env = true
}
