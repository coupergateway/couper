server "scoped jwt" {
  api {
    base_path = "/scope"
    access_control = ["scoped_jwt"]
    required_permission = "z"
    endpoint "/foo" {
      required_permission = {
        get = ""
        post = "foo"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      required_permission = {
        delete = ""
        "*" = "more"
      }
      error_handler "insufficient_permissions" {
        response {
          status = 403
          headers = {
            x-required-permission = request.context.required_permission
          }
        }
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/path/{p}/path" {
      required_permission = request.path_params.p
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/object/{method}" {
      required_permission = {
        (request.path_params.method) = contains(["get", "post"], request.path_params.method) ? "a" : "z"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bad/expression" {
      required_permission = request
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bad/type/number" {
      required_permission = 123
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bad/type/boolean" {
      required_permission = true
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bad/type/tuple" {
      required_permission = ["p1", "p2"]
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bad/type/null" {
      required_permission = null
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/permission-from-api" {
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/role"
    access_control = ["roled_jwt"]
    required_permission = "a"
    endpoint "/foo" {
      required_permission = {
        get = ""
        post = "foo"
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      required_permission = {
        delete = ""
        "*" = "more"
      }
      error_handler "insufficient_permissions" {
        response {
          status = 403
          headers = {
            x-required-permission = request.context.required_permission
          }
        }
      }
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/scope_and_role"
    access_control = ["scoped_and_roled_jwt"]
    endpoint "/foo" {
      required_permission = "d"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      required_permission = "e"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
  }
  api {
    base_path = "/scope_and_role_files"
    access_control = ["scoped_and_roled_jwt_files"]
    endpoint "/foo" {
      required_permission = "d"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
    endpoint "/bar" {
      required_permission = "e"
      response {
        status = 204
        headers = {
          x-granted-permissions = json_encode(request.context.granted_permissions)
        }
      }
    }
  }
}
definitions {
  jwt "scoped_jwt" {
    signature_algorithm = "HS256"
    key = "asdf"
    permissions_claim = "scp"
  }
  jwt "roled_jwt" {
    signature_algorithm = "HS256"
    key = "asdf"
    roles_claim = "rl"
    roles_map = {
      "r1" = ["a", "b"]
    }
  }
  jwt "scoped_and_roled_jwt" {
    signature_algorithm = "HS256"
    key = "asdf"
    permissions_claim = "scp"
    roles_claim = "rl"
    roles_map = {
      "r1" = ["b"]
    }
    permissions_map = {
      a = ["c"]
      b = ["e"] # from role-mapped permission
      c = ["d"]
      d = ["a"] # cycle is ignored
    }
  }
  jwt "scoped_and_roled_jwt_files" {
    signature_algorithm = "HS256"
    key = "asdf"
    permissions_claim = "scp"
    roles_claim = "rl"
    roles_map_file = "roles.json"
    permissions_map_file = "permissions.json"
  }
}
