server {
  access_control = ["in-form_body"]
  endpoint "/with-ac" { # error_type=jwt ... status=403
    response {
      body = request.url
    }
  }

  endpoint "/without-ac" { # error_type=jwt_token_missing ... status=401
    disable_access_control = ["in-form_body"]
    response {
      body = request.url
    }
  }
}

definitions {
  jwt "in-form_body" {
    signature_algorithm = "HS256"
    key = "test123"
    token_value = request.form_body.token[0]
  }
}
