server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/" {
    proxy {
      url = "https://foo.com"

      backend {
        origin = "https://bar.com"
      }
    }

    response {
      body = "pest"
    }
  }
}
