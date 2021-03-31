server "multi-api" {
  api {
    base_path = "/v1"
    endpoint "/xxx" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }

  api {
    base_path = "/v2"
    endpoint "/yyy" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }

  api {
    base_path = "/v3"
    endpoint "/zzz" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }
}
