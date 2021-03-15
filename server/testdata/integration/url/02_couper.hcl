server "url" {
  api {
    endpoint "/" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        backend = "test"
      }
    }
  }
}

definitions {
  backend "test" {
    set_query_params = {
      a = "A"
    }
  }
}
