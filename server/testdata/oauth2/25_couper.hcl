server {
  api "as" {
    base_path = "/as"

    endpoint "/token" {
      response {
        headers = {
          access-token = jwt_sign("at", {})
        }
      }
    }
  }

  api {
    access_control = ["at"]

    endpoint "/**" {
      response {
        status = 204
      }
    }
  }
}

definitions {
  jwt "at" {
    signature_algorithm = "HS256"
    key = "asdf"
    signing_ttl = "60s"

    introspection {
      endpoint = "{{.asOrigin}}/introspect"
      ttl = "{{.ttl}}"
    }
  }
}
