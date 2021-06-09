server "no-gz" {
  hosts      = ["example.org:9898"]
  error_file = "./../integration/server_error.html"

  endpoint "/0" {
    response {
      status = 200
    }
  }

  endpoint "/59" {
    response {
      status = 200
      body = "11111111112222222222333333333344444444445555555555123456789"
    }
  }

  endpoint "/60" {
    response {
      status = 200
      body = "111111111122222222223333333333444444444455555555551234567890"
    }
  }

  endpoint "/x" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/pdf"
    }
  }
}
