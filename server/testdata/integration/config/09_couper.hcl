server "scoped jwt" {
  api {
    base_path = "/scope"
    access_control = ["scoped_jwt"]
    beta_required_permission = "z"
    endpoint "/foo" {
      beta_required_permission = {
        get = ""
        post = "foo"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      beta_required_permission = {
        delete = ""
        "*" = "more"
      }
      error_handler "beta_insufficient_permissions" {
        response {
          status = 403
          headers = {
            x-required-permission = request.context.beta_required_permission
          }
        }
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/path/{p}/path" {
      beta_required_permission = request.path_params.p
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/object/{method}" {
      beta_required_permission = {
        (request.path_params.method) = contains(["get", "post"], request.path_params.method) ? "a" : "z"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bad/expression" {
      beta_required_permission = request
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bad/type/number" {
      beta_required_permission = 123
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bad/type/boolean" {
      beta_required_permission = true
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bad/type/tuple" {
      beta_required_permission = ["p1", "p2"]
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bad/type/null" {
      beta_required_permission = null
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/permission-from-api" {
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/role"
    access_control = ["roled_jwt"]
    beta_required_permission = "a"
    endpoint "/foo" {
      beta_required_permission = {
        get = ""
        post = "foo"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      beta_required_permission = {
        delete = ""
        "*" = "more"
      }
      error_handler "beta_insufficient_permissions" {
        response {
          status = 403
          headers = {
            x-required-permission = request.context.beta_required_permission
          }
        }
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/scope_and_role"
    access_control = ["scoped_and_roled_jwt"]
    endpoint "/foo" {
      beta_required_permission = "d"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      beta_required_permission = "e"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/scope_and_role_files"
    access_control = ["scoped_and_roled_jwt_files"]
    endpoint "/foo" {
      beta_required_permission = "d"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      beta_required_permission = "e"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.beta_granted_permissions)
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
    beta_permissions_claim = "scp"
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
    beta_permissions_claim = "scp"
    beta_roles_claim = "rl"
    beta_roles_map = {
      "r1" = ["b"]
    }
    beta_permissions_map = {
      a = ["c"]
      b = ["e"] # from role-mapped permission
      c = ["d"]
      d = ["a"] # cycle is ignored
    }
  }
  jwt "scoped_and_roled_jwt_files" {
    header = "authorization"
    signature_algorithm = "HS256"
    key = "asdf"
    beta_permissions_claim = "scp"
    beta_roles_claim = "rl"
    beta_roles_map_file = "testdata/integration/config/roles.json"
    beta_permissions_map_file = "testdata/integration/config/permissions.json"
  }
}
