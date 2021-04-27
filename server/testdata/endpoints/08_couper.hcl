server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/pdf" {
    request "pdf" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    response {
      headers = {
        x-body = backend_responses.default.body
      }
      body = backend_responses.pdf.body
    }
  }
}
