server "simple" {
  error_file = "./../server_error.html"

  endpoint "/200" {
    response {}
  }

  endpoint "/200/{body}" {
    response {
      json_body = { query: request.path_params.body }
    }
  }

  endpoint "/204" {
    response {
      status = 204
    }
  }

  endpoint "/301" {
    response {
      status = 301
      headers = {
        Location = "https://couper.io/"
      }
    }
  }
}
