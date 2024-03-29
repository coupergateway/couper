server {
  endpoint "/in-form_body" {
    access_control = ["in-form_body"]
    response {
      body = request.url
    }
  }

  endpoint "/in-json_body" {
    access_control = ["in-json_body"]
    response {
      body = request.url
    }
  }

  endpoint "/in-body" {
    access_control = ["in-body"]
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

  jwt "in-json_body" {
    signature_algorithm = "HS256"
    key = "test123"
    token_value = request.json_body.token
  }

  jwt "in-body" {
    signature_algorithm = "HS256"
    key = "test123"
    token_value = request.body
  }
}
