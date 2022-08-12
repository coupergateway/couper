environment "test" {
  server {
    endpoint "/test" {
      environment "test" "foo" "bar" {
        set_response_headers = {
          X-Test-Env = couper.environment
        }
      }
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
