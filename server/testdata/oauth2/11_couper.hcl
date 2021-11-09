server "oauth-client" {
  api {
    endpoint "/rs1" {
      proxy {
        backend = "rs1"
      }
    }
    endpoint "/rs2" {
      proxy {
        backend = "rs2"
      }
    }
  }
  endpoint "/oauth1/redir" {
    access_control = ["ac-oauth-1"]
    response {
      json_body = request.context.ac-oauth-1
    }
  }
  endpoint "/oauth2/redir" {
    access_control = ["ac-oauth-2"]
    response {
      json_body = request.context.ac-oauth-2
    }
  }
  endpoint "/oidc1/redir" {
    access_control = ["ac-oidc-1"]
    response {
      json_body = request.context.ac-oidc-1
    }
  }
  endpoint "/oidc2/redir" {
    access_control = ["ac-oidc-2"]
    response {
      json_body = request.context.ac-oidc-2
    }
  }
}
definitions {
  # with referenced backend
  beta_oauth2 "ac-oauth-1" {
    authorization_endpoint = "{{.asOrigin}}/auth"
    token_endpoint = "{{.asOrigin}}/token"
    backend = "as"
    client_id = "foo"
    client_secret = "etbinbp4in"
    grant_type = "authorization_code"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
    redirect_uri = "http://localhost:8080/oauth/redir"
  }
  # with inline backend
  beta_oauth2 "ac-oauth-2" {
    authorization_endpoint = "{{.asOrigin}}/auth"
    token_endpoint = "{{.asOrigin}}/token"
    backend {
      origin = "{{.asOrigin}}"
      add_request_headers = {
        x-sub = "myself"
      }
    }
    client_id = "foo"
    client_secret = "etbinbp4in"
    grant_type = "authorization_code"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
    redirect_uri = "http://localhost:8080/oauth/redir"
  }

  # with referenced backend
  beta_oidc "ac-oidc-1" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    backend = "as"
    client_id = "foo"
    client_secret = "etbinbp4in"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
    redirect_uri = "http://localhost:8080/oidc/redir"
  }
  # with inline backend
  beta_oidc "ac-oidc-2" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    configuration_ttl = "1h"
    backend {
      origin = "{{.asOrigin}}"
      add_request_headers = {
        x-sub = "myself"
      }
    }
    client_id = "foo"
    client_secret = "etbinbp4in"
    verifier_method = "ccm_s256"
    verifier_value = request.cookies.pkcecv
    redirect_uri = "http://localhost:8080/oidc/redir"
  }

  # backends for resource server
  # with referenced backend
  backend "rs1" {
    origin = "{{.rsOrigin}}"
    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      backend = "as"
      client_id = "foo"
      client_secret = "etbinbp4in"
      grant_type = "client_credentials"
    }
  }
  # with inline backend
  backend "rs2" {
    origin = "{{.rsOrigin}}"
    oauth2 {
      token_endpoint = "{{.asOrigin}}/token"
      backend {
        origin = "{{.asOrigin}}"
        add_request_headers = {
          x-sub = "myself"
        }
      }
      client_id = "foo"
      client_secret = "etbinbp4in"
      grant_type = "client_credentials"
    }
  }

  # backend for authorization server
  backend "as" {
    origin = "{{.asOrigin}}"
    add_request_headers = {
      x-sub = "myself"
    }
  }

}
