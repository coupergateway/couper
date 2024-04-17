server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/pdf" {
    request "pdf" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    response {
      headers = {
        x-body = substr(default(backend_responses.pdf.body, "n/a") , 0, 8)
      }
      body = backend_responses.pdf.body
    }
  }

  endpoint "/pdf-proxy" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }
  }

  endpoint "/post" {
    request "a" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      body = request.body
    }

    request "b" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      body = request.body
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      set_request_headers = {
        x-body = request.body
      }
    }
  }
}
