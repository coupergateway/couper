server "multi-api" {
  api {
    endpoint "/xxx" {
      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  api {
    endpoint "/yyy" {
      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }

  api {
    endpoint "/zzz" {
      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
      }
    }
  }
}
