server "v-server1" {
  hosts = ["*", "example.com", "example.org:9876"]
  error_file = "./../server_error.html"

  files {
    document_root = "./htdocs_01"
  }

  spa {
    bootstrap_file = "01_app.html"
    paths = ["/spa1/**"]
  }

  api {
    base_path = "/api"
    error_file = "./../api_error.json"

    endpoint "/" {
      proxy {
        backend = "anything"
      }
    }
  }
}

server "v-server2" {
  hosts = ["couper.io", "example.net:9876"]

  error_file = "./../server_error.html"

  files {
    document_root = "./htdocs_02"
  }

  spa {
    bootstrap_file = "02_app.html"
    paths = ["/spa2/**"]
  }

  api {
    base_path = "/api/v2"
    error_file = "./../api_error.json"

    endpoint "/" {
      proxy {
        backend = "anything"
      }
    }
  }
}

server "v-server3" {
  hosts = ["v-server3.com"]
  error_file = "./../server_error.html"

  files {
    document_root = "./htdocs_03"
  }
}

definitions {
  # backend origin within a definition block gets replaced with the integration test "anything" server.
  backend "anything" {
    path = "/anything"
    origin = env.COUPER_TEST_BACKEND_ADDR
  }
}
