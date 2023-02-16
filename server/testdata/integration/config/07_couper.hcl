server "ac" {
  access_control = ["BA", "JWT", "OAuth2", "OIDC", "SAML"]
}

definitions {
  basic_auth "BA" {
    error_handler {
      response {
        status = 500
      }
    }
  }

  beta_oauth2 "OAuth2" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile email"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    verifier_method = "ccm_s256"
    verifier_value = "not_used_here"

    error_handler {
      response {
        status = 500
      }
    }
  }

  oidc "OIDC" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    client_id = "foo"
    client_secret = "etbinbp4in"
    redirect_uri = "http://localhost:8080/cb" # value is not checked
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.cookies.nnc

    error_handler {
      response {
        status = 500
      }
    }
  }

  jwt "JWT" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"

    error_handler {
      response {
        status = 500
      }
    }
  }

  saml "SAML" {
    idp_metadata_file = "../accesscontrol/testdata/idp-metadata.xml"
    sp_acs_url = "http://www.examle.org/saml/acs"
    sp_entity_id = "my-sp-entity-id"

    error_handler {
      response {
        status = 500
      }
    }
  }
}
