server "concurrent-requests" {
  api {
    endpoint "/**" {
      proxy {
        backend = "be"
      }
    }
  }
}

definitions {
  backend "be" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    openapi {
      file = "01_schema.yaml"
    }
  }
}
