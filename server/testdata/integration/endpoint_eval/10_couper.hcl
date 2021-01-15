server "api" {
  api {
    endpoint "/" {
      backend = "anything"
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    origin = env.COUPER_TEST_BACKEND_ADDR

    set_query_params = {
      space = "a b+c"
    }
  }
}
