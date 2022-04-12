server "api" {
  error_file = "./../integration/server_error.html"

  endpoint "/" {
    proxy {
      url = "/?delay=5s"
      backend {
        origin = "${env.COUPER_TEST_BACKEND_ADDR}"
        timeout = "1s"
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
