server {
  hosts = ["*:8080"]

  cors {
    allowed_origins = ["origin-1"]
    allow_credentials = true
  }

  files {
    document_root = "./htdocs"
    custom_log_fields = {
      from = "base"
    }
  }

  endpoint "/free" {
    response {
      status = 400
    }
  }

  api "ERR" {
    base_path = "/errors"

    endpoint "/" {
      request {
        url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
        expected_status = 404
      }
    }

    error_handler "unexpected_status * insufficient_permissions" {
      response {
        status = 405
      }
    }
  }

  api "API" {
    base_path = "/api-1"

    error_handler "insufficient_permissions" {
      response {
        status = 401
      }
    }

    endpoint "/" {
      proxy {
        url = "https://couper.io/"
      }
    }
  }

  api {
    base_path = "/api-2"

    error_handler {
      response {
        status = 401
      }
    }
  }
}

server "ServerA" {
  hosts = ["*:8081"]

  cors {
    allowed_origins = ["origin-a"]
  }

  api {
    endpoint "/" {
      response {
        body = "OK"
      }
    }
  }
}

server "ServerB" {
  hosts = ["*:8082"]

  api {
    endpoint "/" {
      response {
        body = "OK"
      }
    }
  }
}

server "ServerC" {
  hosts = ["*:8083"]
}
