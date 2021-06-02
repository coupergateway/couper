server "devel" {
  set_response_headers = {
    x-server = "true"
  }

  files {
    document_root = "htdocs"
    error_file = "./../server_error.html"

    set_response_headers = {
      x-files = "true"
    }
  }
  spa {
    bootstrap_file = "./app/bootstrap.html"
    paths = ["/app/**"]

    set_response_headers = {
      x-spa = "true"
    }
  }

  api {
    set_response_headers = {
        x-api = "true"
    }

    endpoint "/api/**" {
      proxy {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
      }

      set_response_headers = {
        x-endpoint = "true"
      }
    }
  }
}
