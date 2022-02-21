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

  api "API" {
    base_path = "/api-1"

    error_handler "beta_insufficient_scope" {
      response {
        status = 401
      }
    }

    endpoint "/" {
      proxy {
        url = "https://example.com"
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
}

server "ServerB" {
  hosts = ["*:8082"]
}

server "ServerC" {
  hosts = ["*:8083"]
}
