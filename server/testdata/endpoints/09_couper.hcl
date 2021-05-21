server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/" {
    proxy {
      # should fail
      url = "https://foo.com"

      backend {
        origin = "https://bar.com"

        # should not run
        set_response_headers = {
          x-backend = 1
        }
      }
    }

    # should not run
    set_response_headers = {
      x-endpoint = 1
    }

    # should not run
    response {
      body = "pest"
    }
  }
}
