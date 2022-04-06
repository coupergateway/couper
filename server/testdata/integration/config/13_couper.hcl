server {
  endpoint "/" {
    proxy {
      backend = "ref"
    }
  }

  endpoint "/granted" {
    proxy {
      backend "ref" {
        basic_auth = "peter:pan"
      }
    }
  }
}

definitions {
  backend "ref" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
  }
}

settings {
  no_proxy_from_env = true
}
