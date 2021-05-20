server "form-params" {
  error_file = "./../server_error.html"

  endpoint "/" {
    proxy {
      backend = "anything"
    }

    set_form_params = {
      a = "A"
      b = "B"
      c = ["C 1", "C 2"]
    }

    add_form_params = {
      d = "D"
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
