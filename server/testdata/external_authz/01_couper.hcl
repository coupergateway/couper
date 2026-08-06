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
        status = request.json_body.subject.id == "valid" ? 200 : (request.json_body.subject.type == "anonymous" ? 401 : 403)
      }
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url = "http://127.0.0.1:8081/check"
  }
}
