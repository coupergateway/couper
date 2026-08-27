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
        json_body = request.json_body.subject.id == "valid" ? { decision = true } : (
          request.json_body.subject.type == "anonymous"
          ? { decision = false, context = { www_authenticate = "Bearer error=\"invalid_token\"" } }
          : { decision = false }
        )
      }
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url = "http://127.0.0.1:8081/check"
  }
}
