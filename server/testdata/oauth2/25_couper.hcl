server {
  api "as" {
    base_path = "/as"

    endpoint "/token" { # not proper OAuth2 token endpoint, but token is easier to extract from header
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

    beta_introspection {
      endpoint = "{{.asOrigin}}/introspect"
      ttl = "{{.ttl}}"
      client_id = "the_rs"
      client_secret = "the_rs_asdf"
    }
  }
}
