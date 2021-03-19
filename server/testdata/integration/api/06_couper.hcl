server "multi-api" {
  api {
    endpoint "/" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path_prefix = "/${req.headers.x-val}/xxx/"
        }
      }
    }
    endpoint "/uuu/**" {
      path = "/api/**"
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path_prefix = "/${req.headers.x-val}/xxx/"
        }
      }
    }
    endpoint "/vvv/**" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path = "/api/**"
          path_prefix = "/${req.headers.x-val}/xxx/"
        }
      }
    }
  }

  api {
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
