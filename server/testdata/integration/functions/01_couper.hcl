server "api" {
  api {
    base_path = "/v1"

    endpoint "/merge" {
      response {
        headers = {
          x-merged-1 = json_encode(merge({"foo": [1]}, {"foo": [2]}))
          x-merged-2 = json_encode(merge({bar = [3]}, {bar = [4]}))
          x-merged-3 = json_encode(merge(["a"], ["b"]))
        }
      }
    }

    endpoint "/coalesce" {
      response {
        headers = {
          x-coalesce-1 = coalesce(request.path, "default")
          x-coalesce-2 = coalesce(request.cookies.undef, "default")
          x-coalesce-3 = coalesce(request.query.q[1], "default")
          x-coalesce-4 = coalesce(request.cookies.undef, request.query.q[1], "default", request.path)
        }
      }
    }

    endpoint "/default" {
      response {
        headers = {
          x-default-1 = default(request.path, "default")
          x-default-2 = default(request.cookies.undef, "default")
          x-default-3 = default(request.query.q[1], "default")
          x-default-4 = default(request.cookies.undef, request.query.q[1], "default", request.path)
          x-default-5 = "prefix-${default(request.cookies.undef, "default")}" # template expr
          x-default-6 = "${default(request.cookies.undef, "default")}" # template wrap expr
          x-default-7 = default(env.MY_UNSET_ENV, "default")
          x-default-8 = default(request.query.r, ["default-8"])[0]
          x-default-9 = default(request.cookies.undef, "")
          x-default-10 = default(request.cookies.undef, request.cookies.undef)
          x-default-11 = default(request.cookies.undef, 0)
          x-default-12 = default(request.cookies.undef, false)
          x-default-13 = json_encode(default({a = 1}, {}))
          x-default-14 = json_encode(default({a = 1}, {b = 2}))
          x-default-15 = json_encode(default([1, 2], []))
        }
      }
    }

    endpoint "/to_number" {
      response {
        json_body = {
          float-2_34 = to_number("2.34")
          float-_3 = to_number(".3")
          int = to_number("34")
          int-3_ = to_number("3.")
          int-3_0 = to_number("3.0")
          null = to_number(null)
          from-env = to_number(env.PI)
        }
      }
    }

    endpoint "/to_number/string" {
      response {
        json_body = {
          error = to_number("two")
        }
      }
    }

    endpoint "/to_number/bool" {
      response {
        json_body = {
          error = to_number(true)
        }
      }
    }

    endpoint "/to_number/tuple" {
      response {
        json_body = {
          error = to_number([1])
        }
      }
    }

    endpoint "/to_number/object" {
      response {
        json_body = {
          error = to_number({a = 1})
        }
      }
    }

    endpoint "/contains" {
      response {
        headers = {
          x-contains-1 = contains(["a", "b"], "a") ? "yes" : "no"
          x-contains-2 = contains(["a", "b"], "c") ? "yes" : "no"
          x-contains-3 = contains([0, 1], 0) ? "yes" : "no"
          x-contains-4 = contains([0, 1], 2) ? "yes" : "no"
          x-contains-5 = contains([0.1, 1.1], 0.1) ? "yes" : "no"
          x-contains-6 = contains([0.1, 1.1], 0.10000000001) ? "yes" : "no"
          x-contains-7 = contains([{a = 1, aa = {aaa = 1}}, {b = 2}], {a = 1, aa = {aaa = 1}}) ? "yes" : "no"
          x-contains-8 = contains([{a = 1}, {b = 2}], {c = 3}) ? "yes" : "no"
          x-contains-9 = contains([[1,2], [3,4]], [1,2]) ? "yes" : "no"
          x-contains-10 = contains([[1,2], [3,4]], [5,6]) ? "yes" : "no"
          x-contains-11 = contains(["3.14159", "42"], env.PI) ? "yes" : "no"
        }
      }
    }

    endpoint "/length" {
      response {
        headers = {
          x-length-1 = length([0, 1]) # tuple
          x-length-2 = length([])
          x-length-3 = length(split(",", "0,1,2,3,4")) # list
          x-length-4 = length(request.headers) # map
        }
      }
    }

    endpoint "/length/object" {
      response {
        headers = {
          error = length({a = 1})
        }
      }
    }

    endpoint "/length/string" {
      response {
        headers = {
          error = length("abcde")
        }
      }
    }

    endpoint "/length/null" {
      response {
        headers = {
          error = length(null)
        }
      }
    }

    endpoint "/join" {
      response {
        headers = {
          x-join-1 = join("-",[0, 1],["a","b"],[3,"c"],[1.234],[true,false])
          x-join-2 = "|${join("-",[])}|"
          x-join-3 = join("-", split(",", "0,1,2,3,4"))
        }
      }
    }

    endpoint "/keys" {
      response {
        headers = {
          x-keys-1 = json_encode(keys({a = 1, c = 2, b = {d = 3}}))
          x-keys-2 = json_encode(keys({}))
          x-keys-3 = json_encode(keys(request.headers))
        }
      }
    }

    endpoint "/set_intersection" {
      response {
        headers = {
          x-set_intersection-1 = json_encode((set_intersection([1,3])))
          x-set_intersection-2 = json_encode(set_intersection([0,1,2,3], [1,3]))
          x-set_intersection-3 = json_encode(set_intersection([1,3],[0,1,2,3]))
          x-set_intersection-4 = json_encode(set_intersection([1,3],[1,3]))
          x-set_intersection-5 = json_encode(set_intersection([0,1,2,3], [1,3], [3,5]))
          x-set_intersection-6 = json_encode(set_intersection([0,1,2,3], [3,4,5]))
          x-set_intersection-7 = json_encode(set_intersection([0,1,2,3],[4,5]))
          x-set_intersection-8 = json_encode(set_intersection([0,1,2,3],[]))
          x-set_intersection-9 = json_encode(set_intersection([],[1,3]))
          x-set_intersection-10 = json_encode(set_intersection([0,1,2,3], [1,4], [3,5]))
          x-set_intersection-11 = json_encode(set_intersection([1.1,2.2,3.3], [2.2,4.4]))
          x-set_intersection-12 = json_encode(set_intersection(["a","b","c","d"], ["b","d","e"]))
          x-set_intersection-13 = json_encode(set_intersection([true,false], [true]))
          x-set_intersection-14 = json_encode(set_intersection([{a=1},{b=2}], [{a=1},{c=3}]))
          x-set_intersection-15 = json_encode(set_intersection([[1,2],[3,4]], [[1,2],[5,6]]))
        }
      }
    }

    endpoint "/lookup" {
      response {
        headers = {
          x-lookup-1 = lookup({a = "1"}, "a", "default")
          x-lookup-2 = lookup({a = "1"}, "b", "default")
          x-lookup-3 = lookup(request.headers, "user-agent", "default")
          x-lookup-4 = lookup(request.headers, "content-type", "default")
        }
      }
    }

    endpoint "/lookup/inputMap-null" {
      response {
        headers = {
          error = lookup(null, "a", "default")
        }
      }
    }

    endpoint "/trim" {
      response {
        headers = {
          x-trim = trim(" \tfoo \tbar \t")
        }
      }
    }

    endpoint "/can" {
      response {
        headers = {
          x-can = json_encode({ for k in ["not_there", "method", "path"] : k => request[k] if can(request[k]) })
        }
      }
    }
  }
}

defaults {
  environment_variables = {
    PI = "3.14159"
  }
}
