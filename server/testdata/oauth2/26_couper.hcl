server "rs" {
  hosts = ["*:8080"]

  api {
    endpoint "/csb" {
      access_control = ["at_in_csb"]

      response {
        status = 204
      }
    }

    endpoint "/csp" {
      access_control = ["at_in_csp"]

      response {
        status = 204
      }
    }

    endpoint "/csj" {
      access_control = ["at_in_csj"]

      response {
        status = 204
      }
    }

    endpoint "/pkj" {
      access_control = ["at_in_pkj"]

      response {
        status = 204
      }
    }
  }
}

definitions {
  # for rs
  jwt "at_in_csb" {
    signature_algorithm = "HS256"
    key = "asdf"

    beta_introspection {
      endpoint = "http://1.1.1.1:9999/introspect/csb"
      ttl = "0s"
      client_id = "the_rs"
      client_secret = "the_rs_asdf"
    }
  }

  jwt "at_in_csp" {
    signature_algorithm = "HS256"
    key = "asdf"

    beta_introspection {
      endpoint = "http://1.1.1.1:9999/introspect/csp"
      endpoint_auth_method = "client_secret_post"
      ttl = "0s"
      client_id = "the_rs"
      client_secret = "the_rs_asdf"
    }
  }

  jwt "at_in_csj" {
    signature_algorithm = "HS256"
    key = "asdf"

    beta_introspection {
      endpoint = "http://1.1.1.1:9999/introspect/csj"
      endpoint_auth_method = "client_secret_jwt"
      ttl = "0s"
      client_id = "the_rs"
      client_secret = "the_rs_asdf"

      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }

  jwt "at_in_pkj" {
    signature_algorithm = "HS256"
    key = "asdf"

    beta_introspection {
      endpoint = "http://1.1.1.1:9999/introspect/pkj"
      endpoint_auth_method = "private_key_jwt"
      ttl = "0s"
      client_id = "the_rs"

      jwt_signing_profile {
        signature_algorithm = "RS256"
        key_file = "./pkcs8.key"
        ttl = "10s"
      }
    }
  }

  # for as
  jwt_signing_profile "at" {
    signature_algorithm = "HS256"
    key = "asdf"
    ttl = "60s"
  }

  jwt "at_from_fb" {
    token_value = request.form_body.token[0]
    signature_algorithm = "HS256"
    key = "asdf"
  }

  basic_auth "ba_csb" {
    user = "the_rs"
    password = "the_rs_asdf"
  }

  jwt "jwt_csj" {
    signature_algorithm = "HS256"
    key = "the_rs_asdf"
    token_value = request.form_body.client_assertion[0]
    claims = {
      iss = "the_rs"
      sub = "the_rs"
    }
    required_claims = ["exp", "iat", "jti"]
  }

  jwt "jwt_pkj" {
    signature_algorithm = "RS256"
    key_file = "./certificate.pem"
    token_value = request.form_body.client_assertion[0]
    claims = {
      iss = "the_rs"
      sub = "the_rs"
    }
    required_claims = ["exp", "iat", "jti"]
  }
}

server "as" {
  hosts = ["*:9999"]

  api {
    endpoint "/token" { # not proper OAuth2 token endpoint, but token is easier to extract from header
      response {
        headers = {
          access-token = jwt_sign("at", {})
        }
      }
    }

    endpoint "/introspect/csb" {
      access_control = ["ba_csb", "at_from_fb"]

      response {
        json_body = {active: true}
      }
    }

    endpoint "/introspect/csp" {
      access_control = ["at_from_fb"]

      response {
        status = request.form_body.client_id[0] == "the_rs" && request.form_body.client_secret[0] == "the_rs_asdf" ? 200 : 401
        json_body = {active: true}
      }
    }

    endpoint "/introspect/csj" {
      access_control = ["jwt_csj", "at_from_fb"]

      response {
        json_body = {active: true}
      }
    }

    endpoint "/introspect/pkj" {
      access_control = ["jwt_pkj", "at_from_fb"]

      response {
        json_body = {active: true}
      }
    }
  }
}
