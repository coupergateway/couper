server "url" {
  api {
    endpoint "/" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything?x=y"
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
