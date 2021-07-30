server "oidc-functions" {
  endpoint "/pkce" {
    response {
      headers = {
        x-hv = internal_oauth_hashed_verifier()
        x-au-pkce = beta_oauth_authorization_url("ac-pkce")
      }
    }
  }
  endpoint "/csrf" {
    response {
      headers = {
        x-hv = internal_oauth_hashed_verifier()
        x-au-nonce = beta_oauth_authorization_url("ac-nonce")
      }
    }
  }
  endpoint "/default" {
    response {
      headers = {
        x-hv = internal_oauth_hashed_verifier()
        x-au-default = beta_oauth_authorization_url("ac-default")
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
    verifier_method = "ccm_s256"
    verifier_value = "not_used_here"
  }
  beta_oidc "ac-nonce" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    ttl = "1h"
    scope = "profile"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    verifier_method = "nonce"
    verifier_value = "not_used_here"
  }
  beta_oidc "ac-default" {
    configuration_url = "{{.asOrigin}}/.well-known/openid-configuration"
    ttl = "1h"
    scope = "profile email address"
    redirect_uri = "http://localhost:8085/oidc/callback"
    client_id = "foo"
    client_secret = "5eCr3t"
    verifier_value = "not_used_here"
  }
}
