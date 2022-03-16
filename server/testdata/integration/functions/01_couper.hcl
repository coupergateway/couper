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
  }
}

defaults {
  environment_variables = {
    PI = "3.14159"
  }
}
