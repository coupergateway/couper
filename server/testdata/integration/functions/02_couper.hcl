server "oauth-functions" {
  endpoint "/pkce" {
    response {
      headers = {
        x-v-1 = beta_oauth_verifier()
        x-v-2 = beta_oauth_verifier()
        x-hv = internal_oauth_hashed_verifier()
        x-au-pkce = oauth2_authorization_url("ac-pkce")
        x-au-pkce-rel = oauth2_authorization_url("ac-pkce-relative")
      }
    }
  }

  endpoint "/csrf" {
    response {
      headers = {
        x-hv = internal_oauth_hashed_verifier()
        x-au-state = oauth2_authorization_url("ac-state")
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
    verifier_method = "ccm_s256"
    verifier_value = "not_used_here"
  }

  beta_oauth2 "ac-pkce-relative" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile email"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    verifier_method = "ccm_s256"
    verifier_value = "not_used_here"
  }

  beta_oauth2 "ac-state" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    verifier_method = "state"
    verifier_value = "not_used_here"
  }
}
