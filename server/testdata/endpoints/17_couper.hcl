server {
  endpoint "/in-form_body" {
    access_control = ["in-form_body"]
    response {
      body = request.url
    }
  }

  endpoint "/without-ac" {
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
