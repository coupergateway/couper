server "multi-api" {
  api {
    base_path = "/v1"

    endpoint "/" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path_prefix = "/${request.headers.x-val}/xxx/"
        }
      }
    }
    endpoint "/vvv/**" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path = "/api/**"
          path_prefix = "/${request.headers.x-val}/xxx/"
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
          path_prefix = "/"
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
          path_prefix = "/zzz"
        }
      }
    }
  }
}
