server "form-params" {
  error_file = "./../server_error.html"

  endpoint "/" {
    proxy {
      backend = "anything"
    }

    remove_form_params = ["x"]
  }
}

definitions {
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
