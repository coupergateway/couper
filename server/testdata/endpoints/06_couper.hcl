server "request-limit" {
  error_file = "./../integration/server_error.html"

  api {
    error_file = "./../integration/api_error.json"

    endpoint "/" {
      access_control = ["BA"]
      request_body_limit = "1"
      response {
        status = 204
        body = req.json_body
      }
    }
  }
}

definitions {
  basic_auth "BA" {
    password = "qwertz"
  }
}

