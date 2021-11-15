server "client" {
  log_fields = {
    server = true
    l1 = "server"
    l2 = ["server"]
    l3 = ["server"]
    l4 = ["server"]
  }

  api {
    log_fields = {
      api = true
      l1 = "api"
      l2 = ["api"]
      l3 = null
      l4 = null
    }

    endpoint "/" {
      log_fields = {
        endpoint = true
        l1 = "endpoint"
        l2 = ["endpoint"]
        l3 = ["endpoint"]
      }

      response {
        json_body = request
      }
    }
  }
}