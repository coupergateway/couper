server "concurrent-requests" {
  api {
    endpoint "/**" {
      proxy {
        backend = "test-be"
      }
    }
  }
}

definitions {
  backend "test-be" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    openapi {
      file = "01_schema.yaml"
    }
  }
}
