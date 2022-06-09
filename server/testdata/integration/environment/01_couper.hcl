environment "test" {
  server {
    endpoint "/test" {
      proxy {
        environment "test" {
          url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        }
        environment "prod" {
          access_control = ["auth"]
          url = "${env.COUPER_TEST_BACKEND_ADDR}"
        }
      }
    }
  }
}

environment "prod" {
  server {}
}
