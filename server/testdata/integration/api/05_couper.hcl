server "multi-api" {
  api {
    endpoint "/xxx" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }

  api {
    endpoint "/yyy" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }

  api {
    endpoint "/zzz" {
      proxy {
        backend {
          origin = env.COUPER_TEST_BACKEND_ADDR
        }
      }
    }
  }
}
