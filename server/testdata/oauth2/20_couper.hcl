server {
  hosts = ["*:8080"]

  api {
    endpoint "/csj" {
      proxy {
        backend = "csj"
      }
    }

    endpoint "/csj_error" {
      proxy {
        backend = "csj_error"
      }
    }

    endpoint "/pkj" {
      proxy {
        backend = "pkj"
      }
    }

    endpoint "/pkj_error" {
      proxy {
        backend = "pkj_error"
      }
    }
  }
}

definitions {
  backend "csj" {
    origin = "{{.rsOrigin}}"

    oauth2 {
      token_endpoint = "http://1.1.1.1:9999/token/csj"
      grant_type = "client_credentials"
      client_id = "my_clid"
      client_secret = "my_cls"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }

  backend "csj_error" {
    origin = "{{.rsOrigin}}"

    oauth2 {
      token_endpoint = "http://1.1.1.1:9999/token/csj/error"
      grant_type = "client_credentials"
      client_id = "my_clid"
      client_secret = "my_cls"
      token_endpoint_auth_method = "client_secret_jwt"
      jwt_signing_profile {
        signature_algorithm = "HS256"
        ttl = "10s"
      }
    }
  }

  backend "pkj" {
    origin = "{{.rsOrigin}}"

    oauth2 {
      token_endpoint = "http://1.1.1.1:9999/token/pkj"
      grant_type = "client_credentials"
      client_id = "my_clid"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        key_file = "./testdata/oauth2/pkcs8.key"
        signature_algorithm = "RS256"
        ttl = "10s"
        claims = {
          aud = "some explicit value"
        }
      }
    }
  }

  backend "pkj_error" {
    origin = "{{.rsOrigin}}"

    oauth2 {
      token_endpoint = "http://1.1.1.1:9999/token/pkj/error"
      grant_type = "client_credentials"
      client_id = "my_clid"
      token_endpoint_auth_method = "private_key_jwt"
      jwt_signing_profile {
        key_file = "./testdata/oauth2/pkcs8.key"
        signature_algorithm = "RS256"
        ttl = "10s"
      }
    }
  }

  jwt "csj" {
    token_value = request.form_body.client_assertion[0]
    signature_algorithm = "HS256"
    key = "my_cls"
    claims = {
      iss = "my_clid"
      sub = "my_clid"
      aud = "http://1.1.1.1:9999/token/csj"
    }
    required_claims = ["iat", "exp", "jti"]
  }

  jwt "csj_error" {
    token_value = request.form_body.client_assertion[0]
    signature_algorithm = "HS256"
    key = "wrong key"
  }

  jwt "pkj" {
    token_value = request.form_body.client_assertion[0]
    signature_algorithm = "RS256"
    key_file = "./testdata/oauth2/certificate.pem"
    claims = {
      iss = "my_clid"
      sub = "my_clid"
      aud = "some explicit value"
    }
    required_claims = ["iat", "exp", "jti"]
  }

  jwt "pkj_error" {
    token_value = request.form_body.client_assertion[0]
    signature_algorithm = "HS256"
    key = "wrong key"
  }
}

server {
  hosts = ["*:9999"]

  api {
    endpoint "/token/csj" {
      access_control = ["csj"]

      response {
        json_body = {
          access_token = "${request.context.csj.iat} ${request.context.csj.exp} ${request.context.csj.jti}"
          expires_in = 60
        }
      }
    }

    endpoint "/token/csj/error" {
      access_control = ["csj_error"]

      response {
        json_body = {
          access_token = "qoebnqeb"
          expires_in = 60
        }
      }
    }

    endpoint "/token/pkj" {
      access_control = ["pkj"]

      response {
        json_body = {
          access_token = "${request.context.pkj.iat} ${request.context.pkj.exp} ${request.context.pkj.jti}"
          expires_in = 60
        }
      }
    }

    endpoint "/token/pkj/error" {
      access_control = ["pkj_error"]

      response {
        json_body = {
          access_token = "qoebnqeb"
          expires_in = 60
        }
      }
    }
  }
}
