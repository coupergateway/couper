server {
  endpoint "/**" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        path = "/**"
      }
    }
  }

  endpoint "/p/**" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        path = "/pb/**"
      }
    }
  }

}
