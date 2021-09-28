server "scoped jwt" {
  api {
    access_control = ["myjwt"]
    beta_scope = "a"
    endpoint "/foo" {
      beta_scope = {
        get = ""
        post = "foo"
      }
      response {
        status = 204
      }
    }
    endpoint "/bar" {
      beta_scope = {
        delete = ""
        "*" = "more"
      }
      response {
        status = 204
      }
    }
  }
}
definitions {
  jwt "myjwt" {
    header = "authorization"
    signature_algorithm = "HS256"
    key = "asdf"
    beta_scope_claim = "scp"
  }
}
