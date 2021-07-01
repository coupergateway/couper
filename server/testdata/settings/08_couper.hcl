server "request-id" {
  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      set_request_headers = {
        Request-ID-From-Var = request.id
      }
    }
  }
}

settings {
  request_id_accept_from_header = "Client-Request-ID"
}
