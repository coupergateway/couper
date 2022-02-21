server {
  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
  }
}

definitions {
  backend "Backend" {
    origin = "https://example.com"
  }

  basic_auth "BA" {
    user     = "U"
    password = "P"
  }
}
