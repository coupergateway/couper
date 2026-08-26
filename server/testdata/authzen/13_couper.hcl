server "protected" {
  hosts = ["*:8080"]

  api {
    # The dynamic required permission replaces the candidate list for this request.
    endpoint "/todos/{action}" {
      access_control      = ["authz"]
      required_permission = "can_${request.path_params.action}"

      response {
        status = 204
        headers = {
          x-asked = request.context.authz.asked
        }
      }
    }

    # A method map resolves per request; a method without an entry falls back to the
    # candidate list, and the permissions control reports it.
    endpoint "/mapped" {
      access_control      = ["authz"]
      required_permission = { GET = "can_read" }

      response {
        status = 204
        headers = {
          x-asked = request.context.authz.asked
        }
      }
    }

    # The wildcard entry covers only the standard methods, mirroring the permissions
    # control: an explicitly allowed custom method must be listed to have a permission.
    endpoint "/wildcard" {
      access_control      = ["authz"]
      allowed_methods     = ["GET", "BREW"]
      required_permission = { "*" = "can_read" }

      response {
        status = 204
        headers = {
          x-asked = request.context.authz.asked
        }
      }
    }

    # request.json_body must stay readable with required_permission set — the expression
    # wrapper must keep the endpoint reachable for the server's buffer-option lookup.
    endpoint "/body" {
      access_control      = ["authz"]
      required_permission = "can_read"

      response {
        status = 204
        headers = {
          x-method = default(request.json_body.method, "-")
        }
      }
    }
  }

  # Endpoints with and without required_permission must not share one api block.
  api {
    # Without a required permission the configured candidates are evaluated.
    endpoint "/plain" {
      access_control = ["authz"]

      response {
        status = 204
        headers = {
          x-asked = request.context.authz.asked
        }
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    # Allows the client request and the can_read permission; echoes the asked action names.
    # The custom log field captures what was actually asked — the only observable batch
    # evidence when the protected endpoint exits through an error (e.g. method not allowed).
    endpoint "/evaluations" {
      custom_log_fields = {
        asked = join(",", [for ev in request.json_body.evaluations : ev.action.name])
      }

      response {
        json_body = {
          evaluations = [for i, e in request.json_body.evaluations : {
            decision = i == 0 ? true : e.action.name == "can_read"
            context = {
              asked = join(",", [for ev in request.json_body.evaluations : ev.action.name])
            }
          }]
        }
      }
    }
  }
}

definitions {
  beta_authzen "authz" {
    url                  = "http://127.0.0.1:8081/evaluations"
    evaluate_permissions = ["can_read", "can_write"]
  }
}
