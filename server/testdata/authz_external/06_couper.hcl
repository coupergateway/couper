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
    endpoint "/check" {
      response {
        json_body = {
          granted_permissions = lookup(request.json_body.client_request.headers, "Authorization", [""])[0] == "Bearer reader" ? ["can_read", "can_list"] : ["can_list"]
        }
      }
    }
  }
}

definitions {
  beta_authz_external "authz" {
    url               = "http://127.0.0.1:8081/check"
    permissions_property = "granted_permissions"
  }
}
