server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/stock" {
      access_control = ["stock_authz"]

      response {
        status = 204
      }
    }

    endpoint "/broken" {
      access_control = ["broken_authz"]

      response {
        status = 204
      }
    }
  }
}

server "authz-service" {
  hosts = ["*:8081"]

  api {
    # A decision point without any Couper convention answers with a flat deny.
    endpoint "/stock" {
      response {
        json_body = { decision = false }
      }
    }

    # Couper failed to authenticate to the decision point. This challenge is addressed to
    # Couper, not to the client.
    endpoint "/broken" {
      response {
        status = 401
        headers = {
          www-authenticate = "Bearer realm=\"pdp.example\""
        }
      }
    }
  }
}

definitions {
  beta_external_authz "stock_authz" {
    url = "http://127.0.0.1:8081/stock"
  }

  beta_external_authz "broken_authz" {
    url = "http://127.0.0.1:8081/broken"
  }
}
