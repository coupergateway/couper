server "scoped" {

  access_control = ["secured"]

  api {
    base_path = "/api"

    required_permission = "read"
    endpoint "/" {
      response {
        status = 204
      }
    }

    endpoint "/pow/" {
      required_permission = {
        post = "power"
      }

      response {
        status = 204
      }

      error_handler "beta_insufficient_permissions" {
        response {
          status = 400
          body = "Not enough power"
        }
      }
    }

    error_handler "beta_insufficient_permissions" {
      response {
        status = 418
      }
    }

  }

  api {
    base_path = "/wildcard1"

    error_handler "beta_insufficient_permissions" {
      response {
        status = 418
        body = "Not enough power"
      }
    }

    endpoint "/" {
      required_permission = "power"

      response {
        status = 204
      }

      error_handler "*" {
        response {
          status = 400
          body = "Not enough power"
        }
      }
    }
  }

  api {
    base_path = "/wildcard2"

    error_handler {
      response {
        status = 418
        body = "Not enough power"
      }
    }

    endpoint "/" {
      required_permission = "power"

      response {
        status = 204
      }

      error_handler "beta_insufficient_permissions" {
        response {
          status = 400
          body = "Not enough power"
        }
      }
    }
  }

  endpoint "/" {
    required_permission = "write"

    response {
      body = "OK"
    }

    error_handler "beta_insufficient_permissions" {
      response {
        status = 418
      }
    }

  }
}

definitions {
  jwt "secured" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "s3cr3t"
    beta_permissions_claim = "scope"
  }
}
