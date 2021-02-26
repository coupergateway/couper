server "api" {
  api {

    backend {
      origin = env.COUPER_TEST_BACKEND_ADDR != "" ? env.COUPER_TEST_BACKEND_ADDR : "http://127.0.0.1:1"
      path = "/unset/by/endpoint"
    }

    endpoint "/endpoint1" {
      proxy {
        backend "anything" {
          path = "/anything"
        }
      }
    }

    endpoint "/endpoint2" {
      path = "/unset/by/local-backend"
      proxy {
        backend "anything" {
          path = "/anything"
        }
      }
    }

    # don't override path
    endpoint "/endpoint3" {
      proxy {
        backend = "anything"
      }
    }

    endpoint "/endpoint4" {
      path = "/unset/here"
      proxy {
        backend "anything" {
          path = "/anything"
        }
      }
    }

  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/unset/by/endpoint"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
