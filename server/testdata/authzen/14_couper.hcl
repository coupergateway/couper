server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        status = 204
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    # A decision point rejecting the evaluation request itself, for example because the
    # asked action does not exist in its model.
    endpoint "/evaluation" {
      response {
        status    = 400
        json_body = {
          code = "validation_error"
        }
      }
    }
  }
}

definitions {
  beta_authzen "authz" {
    url = "http://127.0.0.1:8081/evaluation"

    error_handler "authzen" {
      response {
        status = 403
        json_body = {
          error = "denied"
        }
      }
    }
  }
}
