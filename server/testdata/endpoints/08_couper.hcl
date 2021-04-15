server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/pdf" {
    request "pdf" {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }

    response {
      body = backend_responses.pdf.body
    }
  }
}
