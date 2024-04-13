server "cl" {
  api {
    endpoint "/1" {
      proxy {
        backend = "be"
      }
      set_query_params = {
        B = true
        N = 1
        S = "str"
        Ba = [true, false]
        Na = [1, 2]
        Sa = ["s1", "s2"]
      }
      add_query_params = {
        B2 = false
        N2 = 2
        S2 = "asdf"
        Ba2 = [false, true]
        Na2 = [3, 4]
        Sa2 = ["s3", "s4"]
      }
      set_form_params = {
        B = true
        N = 1
        S = "str"
        Ba = [true, false]
        Na = [1, 2]
        Sa = ["s1", "s2"]
      }
      add_form_params = {
        B2 = false
        N2 = 2
        S2 = "asdf"
        Ba2 = [false, true]
        Na2 = [3, 4]
        Sa2 = ["s3", "s4"]
      }
      set_request_headers = {
        B = true
        N = 1
        S = "str"
        Ba = [true, false]
        Na = [1, 2]
        Sa = ["s1", "s2"]
      }
      add_request_headers = {
        B2 = false
        N2 = 2
        S2 = "asdf"
        Ba2 = [false, true]
        Na2 = [3, 4]
        Sa2 = ["s3", "s4"]
      }
      set_response_headers = {
        B = true
        N = 1
        S = "str"
        Ba = [true, false]
        Na = [1, 2]
        Sa = ["s1", "s2"]
      }
      add_response_headers = {
        B2 = false
        N2 = 2
        S2 = "asdf"
        Ba2 = [false, true]
        Na2 = [3, 4]
        Sa2 = ["s3", "s4"]
      }
    }

    endpoint "/2" {
      request {
        url = "/anything"
        backend = "be"
        headers = {
          B = true
          N = 1
          S = "str"
          Ba = [true, false]
          Na = [1, 2]
          Sa = ["s1", "s2"]
        }
      }
      response {
        headers = {
          B = true
          N = 1
          S = "str"
          Ba = [true, false]
          Na = [1, 2]
          Sa = ["s1", "s2"]
        }
        body = backend_responses.default.body
      }
    }
  }
}

definitions {
  backend "be" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/anything"
  }
}
