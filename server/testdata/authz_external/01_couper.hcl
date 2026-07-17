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
    endpoint "/check" {
      response {
        status = lookup(request.json_body.client_request.headers, "Authorization", [""])[0] == "Bearer valid" ? 200 : (lookup(request.json_body.client_request.headers, "Authorization", [""])[0] == "" ? 401 : 403)
      }
    }
  }
}

definitions {
  beta_authz_external "authz" {
    url = "http://127.0.0.1:8081/check"
  }
}
