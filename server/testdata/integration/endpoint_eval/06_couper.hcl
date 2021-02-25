server "api" {
  api {
    endpoint "/" {
      path = "/yyy"
      proxy {
        backend "anything" {
          path = "/zzz"
        }
      }

      add_query_params = {
        ae = "ae"
      }
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/xxx"
    origin = env.COUPER_TEST_BACKEND_ADDR

    add_query_params = {
      def = "def"
    }
  }
}
