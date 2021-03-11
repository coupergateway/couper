server "multi-api" {
  api {
    endpoint "/" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
          path_prefix = "/xxx/xxx/"
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
