server "logs" {
  files {
    document_root = "./"
    custom_log_fields = {
      files = request.method
    }
  }

  spa {
    bootstrap_file = "./file.html"
    paths = ["/spa/**"]
    custom_log_fields = {
      spa = request.method
    }
  }

  custom_log_fields = {
    server = backend_responses.default.headers.server
  }

  endpoint "/secure" {
    access_control = ["BA"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt-valid" {
    access_control = ["JWT"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt" {
    access_control = ["JWT"]

    proxy {
      backend = "BE"
    }
  }

  endpoint "/jwt-wildcard" {
    access_control = ["JWT-WILDCARD"]

    proxy {
      backend = "BE"
    }
  }

  api {
    custom_log_fields = {
      api = backend_responses.default.headers.server
    }

    endpoint "/" {
      custom_log_fields = {
        endpoint = backend_responses.default.headers.server
      }

      proxy {
        backend "BE" {
          custom_log_fields = {
            bool   = true
            int    = 123
            float  = 1.23
            string = backend_responses.default.headers.server
            req    = request.method

            array = [
              1,
              backend_responses.default.headers.server,
              [
                2,
                backend_responses.default.headers.server
              ],
              {
                x = "X"
              }
            ]

            object = {
              a = "A"
              b = "B"
              c = 123
            }
          }
        }
      }
    }

    endpoint "/backend" {
      proxy {
        backend = "BE"
      }
    }

    endpoint "/oauth2cb" {
      access_control = ["oauth2-regular"]
      proxy {
        backend = "BE"
      }
    }

    endpoint "/oauth2cb-wildcard" {
      access_control = ["oauth2-wildcard"]
      proxy {
        backend = "BE"
      }
    }

    endpoint "/saml-saml2/acs" {
      access_control = ["SSO-saml2"]

      response {
        status = 418
      }
    }

    endpoint "/saml-saml/acs" {
      access_control = ["SSO-saml"]

      response {
        status = 418
      }
    }

    endpoint "/saml-wildcard/acs" {
      access_control = ["SSO-wildcard"]

      response {
        status = 418
      }
    }

    endpoint "/oidc/cb" {
      access_control = ["oidc"]

      response {
        status = 204
      }
    }

    endpoint "/oidc-wildcard/cb" {
      access_control = ["oidc-wildcard"]

      response {
        status = 204
      }
    }
  }

  api {
    endpoint "/error-handler/endpoint" {
      access_control = ["JWT"]

      required_permission = "required"

      response {
        status = 204
      }

      error_handler "insufficient_permissions" {
        custom_log_fields = {
          error_handler = request.method
        }
      }
    }
  }

  endpoint "/standard" {
    request "resolve" {
      backend = "BE"
    }

    proxy {
      backend = "BE"
    }

    custom_log_fields = {
      item-1 = backend_responses.resolve.json_body.JSON.list[0]
      item-2 = backend_responses.default.json_body.JSON.list[0]
    }
  }

  endpoint "/sequence" {
    request "resolve" {
      backend = "BE"
    }

    proxy {
      backend = "BE"
      set_request_headers = {
        x-status = backend_responses.resolve.status
      }
    }

    custom_log_fields = {
      seq-item-1 = backend_responses.resolve.json_body.JSON.list[0]
      seq-item-2 = backend_responses.default.json_body.JSON.list[0]
    }
  }

}

definitions {
  backend "BE" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path   = "/anything"

    custom_log_fields = {
      backend = backend_response.headers.server
    }
  }

  basic_auth "BA" {
    password = "secret"

    error_handler "basic_auth" {
      custom_log_fields = {
        error_handler = request.method
      }
    }
  }

  jwt "JWT" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"

    custom_log_fields = {
      jwt_regular = request.method
    }

    error_handler "jwt_token_missing" "jwt" {
      custom_log_fields = {
        jwt_error = request.method
      }
    }
  }

  jwt "JWT-WILDCARD" {
    header = "Authorization"
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"

    custom_log_fields = {
      jwt_regular = request.method
    }

    error_handler {
      custom_log_fields = {
        jwt_error_wildcard = request.method
      }
    }
  }

  beta_oauth2 "oauth2-regular" {
    grant_type = "authorization_code"
    redirect_uri = "http://localhost:8080/oauth2cb" # value is not checked
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    token_endpoint = "not.checked/token"
    token_endpoint_auth_method = "client_secret_post"
    verifier_method = "ccm_s256"
    verifier_value = request.query.pkcecv
    client_id = "foo"
    client_secret = "etbinbp4in"
    custom_log_fields = {
      oauth2_regular = request.method
    }

    error_handler "oauth2" {
      custom_log_fields = {
        oauth2_error = request.method
      }
    }
  }

  beta_oauth2 "oauth2-wildcard" {
    grant_type = "authorization_code"
    redirect_uri = "http://localhost:8080/oauth2cb-wildcard" # value is not checked
    authorization_endpoint = "https://authorization.server/oauth2/authorize"
    token_endpoint = "not.checked/token"
    token_endpoint_auth_method = "client_secret_post"
    verifier_method = "ccm_s256"
    verifier_value = request.query.pkcecv
    client_id = "foo"
    client_secret = "etbinbp4in"
    custom_log_fields = {
      oauth2_regular = request.method
    }

    error_handler {
      custom_log_fields = {
        oauth2_wildcard_error = request.method
      }
    }
  }

  saml "SSO-saml2" {
    idp_metadata_file = "../../../../accesscontrol/testdata/idp-metadata.xml"
    sp_acs_url = "http://localhost:8080/saml/acs"
    sp_entity_id = "local-test"
    array_attributes = ["memberOf"]

    custom_log_fields = {
      saml_regular = request.method
    }

    error_handler "saml2" {
      custom_log_fields = {
        saml_saml2_error = request.method
      }
    }
  }

  saml "SSO-saml" {
    idp_metadata_file = "../../../../accesscontrol/testdata/idp-metadata.xml"
    sp_acs_url = "http://localhost:8080/saml/acs"
    sp_entity_id = "local-test"
    array_attributes = ["memberOf"]

    custom_log_fields = {
      saml_regular = request.method
    }

    error_handler "saml" {
      custom_log_fields = {
        saml_saml_error = request.method
      }
    }
  }

  saml "SSO-wildcard" {
    idp_metadata_file = "../../../../accesscontrol/testdata/idp-metadata.xml"
    sp_acs_url = "http://localhost:8080/saml/acs"
    sp_entity_id = "local-test"
    array_attributes = ["memberOf"]

    custom_log_fields = {
      saml_regular = request.method
    }

    error_handler {
      custom_log_fields = {
        saml_wildcard_error = request.method
      }
    }
  }

  oidc "oidc" {
    configuration_url = "${env.COUPER_TEST_BACKEND_ADDR}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    client_id = "foo"
    client_secret = "custom-logs-3344"
    redirect_uri = "http://localhost:8080/oidc/cb" # value is not checked
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.query.nnc

    custom_log_fields = {
      oidc_regular = request.method
    }

    error_handler "oauth2" {
      custom_log_fields = {
        oidc_error = request.method
      }
    }
  }

  oidc "oidc-wildcard" {
    configuration_url = "${env.COUPER_TEST_BACKEND_ADDR}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    client_id = "foo"
    client_secret = "custom-logs-3344"
    redirect_uri = "http://localhost:8080/oidc/cb" # value is not checked
    scope = "profile email"
    verifier_method = "nonce"
    verifier_value = request.query.nnc

    custom_log_fields = {
      oidc_regular = request.method
    }

    error_handler {
      custom_log_fields = {
        oidc_wildcard_error = request.method
      }
    }
  }
}
