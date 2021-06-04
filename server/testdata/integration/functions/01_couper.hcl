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
  }
}
