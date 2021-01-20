server "backends" {

  backend "b" {
    add_response_headers = {
      foo = "2"
    }
  }

  api {
    backend "b" {
      origin = env.COUPER_TEST_BACKEND_ADDR
      add_response_headers = {
        foo = "3"
      }
      add_query_params = {
        bar = "2"
      }
    }

    endpoint "/anything" {
      add_query_params = {
        bar = "3"
      }
      backend {
        origin = env.COUPER_TEST_BACKEND_ADDR
        add_response_headers = {
          foo = "4"
        }
        add_query_params = {
          bar = "4"
        }
      }
    }

    endpoint "/" {
      backend "b" {
        add_response_headers = {
          foo = "4"
        }
      }
    }

    endpoint "/get" {
      add_query_params = {
        bar = "3"
      }
      backend "a" {
        add_response_headers = {
          foo = "3"
        }
        add_query_params = {
          bar = "4"
        }
      }
    }
  }
}

definitions {
  backend "b" {
    origin = "http://1.2.3.4"
    set_response_headers = {
      foo = "1"
    }
    set_query_params = {
      bar = "1"
    }
  }
  backend "a" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
    set_response_headers = {
      foo = "1"
    }
    set_query_params = {
      bar = "1"
    }
  }
}

settings {
  no_proxy_from_env = true
}
