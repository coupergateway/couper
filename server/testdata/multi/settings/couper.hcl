server {
  endpoint "/" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
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
