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
    # Answers every boxcarred question in order: the client request itself, then one entry
    # per candidate permission. A reader may read, nobody may write.
    endpoint "/evaluations" {
      response {
        json_body = {
          evaluations = [for e in request.json_body.evaluations : {
            decision = e.action.name == "GET" ? true : (
              e.action.name == "can_read" && request.json_body.subject.id == "reader"
            )
          }]
        }
      }
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url                  = "http://127.0.0.1:8081/evaluations"
    evaluate_permissions = ["can_read", "can_write"]
  }
}
