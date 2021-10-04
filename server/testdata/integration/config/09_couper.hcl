server "scoped jwt" {
  api {
    base_path = "/scope"
    access_control = ["scoped_jwt"]
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
  api {
    base_path = "/role"
    access_control = ["roled_jwt"]
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
  jwt "scoped_jwt" {
    header = "authorization"
    signature_algorithm = "HS256"
    key = "asdf"
    beta_scope_claim = "scp"
  }
  jwt "roled_jwt" {
    header = "authorization"
    signature_algorithm = "HS256"
    key = "asdf"
    beta_role_claim = "rl"
    beta_role_map = {
      "r1" = ["a", "b"]
    }
  }
}
