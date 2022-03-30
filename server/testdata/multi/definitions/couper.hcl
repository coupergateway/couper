server {
  endpoint "/" {
    proxy {
      backend = "Backend"
    }
  }
}

definitions {
  backend "Backend" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/anything"
  }

  basic_auth "BA" {
    user     = "U"
    password = "P"
  }
}
