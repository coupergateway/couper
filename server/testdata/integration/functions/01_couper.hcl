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
  }
}

defaults {
  environment_variables = {
    PI = "3.14159"
  }
}
