server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/todos/{id}" {
      # The jwt block runs first, so its claims name the subject of the callout.
      access_control = ["token", "authz"]

      response {
        status = 204
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    endpoint "/check" {
      response {
        json_body = {
          decision = request.json_body.subject.id == "clark.kent" && request.json_body.resource.id == "/todos/{id}"
          context = {
            seen_subject_type = request.json_body.subject.type
            seen_action       = request.json_body.action.name
            seen_tenant       = request.json_body.context.tenant
          }
        }
      }
    }
  }
}

definitions {
  jwt "token" {
    signature_algorithm = "HS256"
    key                 = "test123"
  }

  beta_authzen "authz" {
    url = "http://127.0.0.1:8081/check"

    subject = {
      type = "identity"
      id   = request.context.token.sub
    }

    context = {
      tenant = "acme"
    }
  }
}
