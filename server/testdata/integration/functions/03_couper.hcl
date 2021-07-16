server "oidc-functions" {
  endpoint "/pkce-ok" {
    response {
      headers = {
        x-cc-s256 = beta_oauth_code_challenge("S256")
        x-au-pkce = beta_oauth_authorization_url("ac-pkce")
      }
    }
  }
  endpoint "/csrf" {
    response {
      headers = {
        x-cht = beta_oauth_hashed_csrf_token()
        x-au-nonce = beta_oauth_authorization_url("ac-nonce")
      }
    }
  }
  endpoint "/pkce-nok" {
    response {
      headers = {
        x-cc-nok = beta_oauth_code_challenge("nok")
      }
    }
  }
}
definitions {
  beta_oidc "ac-pkce" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    ttl = "1h"
    scope = "profile email"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    pkce {
      code_challenge_method = "S256"
      code_verifier_value = "not_used_here"
    }
  }
  beta_oidc "ac-nonce" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    ttl = "1h"
    scope = "profile"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    csrf {
      token_param = "nonce"
      token_value = "not_used_here"
    }
  }
}
