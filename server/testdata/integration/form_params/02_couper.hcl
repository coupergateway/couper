server "form-params" {
  error_file = "./../server_error.html"

  endpoint "/" {
    proxy {
      backend = "anything"
    }
  }
}

definitions {
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
