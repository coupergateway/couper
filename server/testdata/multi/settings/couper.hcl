server {
  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"

      set_response_headers = {
        X: env.X
        Y: env.Y
      }
    }
  }
}

defaults {
  environment_variables = {
    X = "X"
  }
}

settings {
  health_path = "/abc"
  ca_file = "../../integration/files/not-there.pem"
}
