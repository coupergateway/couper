server {
  hosts = ["*:9080"]

  cors {
    allowed_origins = ["origin-2"]
  }

  files {
    document_root = "./www"
    custom_log_fields = {
      from = "override"
    }
  }

  endpoint "/free" {
    response {
      status = 403
    }
  }

  api "ERR" {
    access_control = ["scoped"]
    required_permission = {
      GET = "gimme"
    }
    error_handler "beta_insufficient_permissions" "unexpected_status" "*" {
      response {
        status = 418
      }
    }
  }

  api "API" {
    base_path      = "/api-111"
    access_control = ["foo"]

    error_handler "beta_insufficient_permissions" {
      response {
        status = 415
      }
    }

    endpoint "/" {
      request "r" {
        url = "https://couper.io/"
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

    endpoint "/" {
      response {
        status = 418
      }
    }
  }

  api "NewAPI" {
    base_path = "/api-4"
    endpoint "/ep" {
      response {
        status = 204
      }
    }
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

  jwt "scoped" {
    jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
  }
}
