server "multi-api-host1" {
  hosts = ["couper.io:9898"]
  error_file = "./../server_error.html"

  api {
    base_path = "/v2"
	error_file = "./../api_error.json"

    endpoint "/abc" {
      backend = "anything"
    }
  }
}

server "multi-api-host2" {
  hosts = ["*:9898"]
  error_file = "./../server_error.html"

  api {
    base_path = "/v3"
	error_file = "./../api_error.json"

    endpoint "/def" {
      backend = "anything"
    }
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = "http://anyserver/"
  }
}
