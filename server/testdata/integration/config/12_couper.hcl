server {
  endpoint "/" {
    proxy {
      backend = "ref"
    }
  }

  endpoint "/prefixed" {
    proxy {
      backend "ref" {
        path_prefix = "/my-prefix"
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
