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
