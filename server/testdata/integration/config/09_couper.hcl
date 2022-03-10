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
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
      }
    }
    endpoint "/bar" {
      beta_scope = {
        delete = ""
        "*" = "more"
      }
      response {
        status = 204
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
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
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
      }
    }
    endpoint "/bar" {
      beta_scope = {
        delete = ""
        "*" = "more"
      }
      response {
        status = 204
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
      }
    }
  }
  api {
    base_path = "/scope_and_role"
    access_control = ["scoped_and_roled_jwt"]
    endpoint "/foo" {
      beta_scope = "d"
      response {
        status = 204
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
      }
    }
    endpoint "/bar" {
      beta_scope = "e"
      response {
        status = 204
        headers = {
          x-granted-scope = json_encode(request.context.scopes)
        }
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
    beta_roles_claim = "rl"
    beta_roles_map = {
      "r1" = ["a", "b"]
    }
  }
  jwt "scoped_and_roled_jwt" {
    header = "authorization"
    signature_algorithm = "HS256"
    key = "asdf"
    beta_scope_claim = "scp"
    beta_roles_claim = "rl"
    beta_roles_map = {
      "r1" = ["b"]
    }
    beta_scope_map = {
      a = ["c"]
      b = ["e"] # from role-mapped scope
      c = ["d"]
      d = ["a"] # cycle is ignored
    }
  }
}
