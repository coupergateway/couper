server {
  endpoint "/req/**" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/**"
    }
  }

  endpoint "/req-backend/**" {
    request {
      url = "/**?a=c"
      backend = "relative"
    }
  }

  endpoint "/req-query/**" {
    request {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/**?a=c"
    }
  }

  endpoint "/proxy/**" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/**"
    }
  }

  endpoint "/proxy-query/**" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/**?a=c"
    }
  }

  endpoint "/proxy-backend/**" {
    proxy {
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        path = "/**"
      }
    }
  }

  endpoint "/proxy-backend-rel/**" {
    proxy {
      url = "/**?a=c"
      backend = "relative"
    }
  }

  endpoint "/proxy-backend-path/**" {
    proxy {
      url = "/**?a=c"
      backend = "relative_path_wins"
    }
  }
}

definitions {
  backend "relative" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
  }

  backend "relative_path_wins" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/anything"
  }
}
