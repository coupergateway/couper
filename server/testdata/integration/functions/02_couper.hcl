server "oauth-functions" {
  endpoint "/pkce-ok" {
    response {
      headers = {
        x-cv-1 = oauth_code_verifier()
        x-cv-2 = oauth_code_verifier()
        x-cc-plain = oauth_code_challenge("plain")
        x-cc-s256 = oauth_code_challenge("S256")
        x-ct-1 = oauth_csrf_token()
        x-ct-2 = oauth_csrf_token()
        x-cht = oauth_hashed_csrf_token()
		x-au-pkce = oauth_authorization_url("ac-pkce")
		x-au-state = oauth_authorization_url("ac-state")
      }
    }
  }
  endpoint "/pkce-nok" {
    response {
      headers = {
        x-cc-nok = oauth_code_challenge("nok")
      }
    }
  }
}
definitions {
  oauth2 "ac-pkce" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile email"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    code_challenge_method = "S256"
  }
  oauth2 "ac-state" {
    grant_type = "authorization_code"
    authorization_endpoint = "https://authorization.server/oauth/authorize"
    scope = "openid profile"
    token_endpoint = "https://authorization.server/oauth/token"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    csrf_token_param = "state"
  }
}
