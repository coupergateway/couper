server "oauth-functions" {
  endpoint "/pkce-ok" {
    response {
      headers = {
        x-cv-1 = beta_oauth_code_verifier()
        x-cv-2 = beta_oauth_code_verifier()
        x-cc-plain = beta_oauth_code_challenge("plain")
        x-cc-s256 = beta_oauth_code_challenge("S256")
        x-ct-1 = beta_oauth_csrf_token()
        x-ct-2 = beta_oauth_csrf_token()
        x-cht = beta_oauth_hashed_csrf_token()
        x-au-pkce = beta_oauth_authorization_url("ac-pkce")
        x-au-state = beta_oauth_authorization_url("ac-state")
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
  beta_oauth2 "ac-pkce" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile email"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    pkce {
      code_challenge_method = "S256"
      code_verifier_value = "not_used_here"
    }
  }
  beta_oauth2 "ac-state" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    csrf {
      token_param = "state"
      token_value = "not_used_here"
    }
  }
  beta_oauth2 "ac-nonce" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    csrf {
      token_param = "nonce"
      token_value = "not_used_here"
    }
  }
}
