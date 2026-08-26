server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control      = ["authz"]
      required_permission = "can_read"

      response {
        status = 204
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    # Allows the client request; grants can_read only to the reader subject.
    endpoint "/check" {
      response {
        json_body = {
          evaluations = [for i, e in request.json_body.evaluations : {
            decision = i == 0 ? true : request.json_body.subject.id == "reader"
          }]
        }
      }
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url                  = "http://127.0.0.1:8081/check"
    evaluate_permissions = ["can_read"]
  }
}
