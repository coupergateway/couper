server "request-id" {
  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
  }
}

settings {
  request_id_accept_from_header = "Client-Request-ID"
  request_id_backend_header     = ""
  request_id_client_header      = ""
}
