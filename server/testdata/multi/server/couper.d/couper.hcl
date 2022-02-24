server {
  hosts = ["*:9080"]
    
  cors {
    allowed_origins = ["origin-2"]
  }

  files {
    document_root = "./web"
    custom_log_fields = {
      from = "override"
    }
  }

  endpoint "/free" {
    response {
      status = 401
    }
  }

  api "API" {
    base_path      = "/api-111"
    access_control = ["foo"]

    error_handler "beta_insufficient_scope" {
      response {
        status = 415
      }
    }

    endpoint "/" {
      request "r" {
        url = "https://example.com"
      }

      response {
        status = 204
      }
    }

    endpoint "/other" {
      response {
        status = 200
      }
    }
  }

  api {
    base_path = "/api-3"

    error_handler {
      response {
        status = 415
      }
    }
  }

  api "NewAPI" {
    base_path = "/api-4"
  }
}

server "ServerA" {
  hosts = ["*:9081"]
}

server "ServerB" {
  cors {
    allowed_origins = ["origin-b"]
  }
}

server "ServerD" {
  hosts = ["*:9084"]
}

definitions {
  basic_auth "foo" {
    password = "abc"
  }
}
