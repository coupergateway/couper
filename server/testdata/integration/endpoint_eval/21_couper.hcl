server {
  api {
    # from https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md#for-expressions
    endpoint "/for0" {
      response {
        json_body = [for v in ["a", "b"]: v] # ["a", "b"]
      }
    }
    endpoint "/for1" {
      response {
        json_body = [for i, v in ["a", "b"]: i] # [0, 1]
      }
    }
    endpoint "/for2" {
      response {
        json_body = {for i, v in ["a", "b"]: v => i} # {a = 0, b = 1}
      }
    }
    endpoint "/for3" {
      response {
        json_body = {for i, v in ["a", "a", "b"]: v => i...} # {a = [0, 1], b = [2]}
      }
    }
    endpoint "/for4" {
      response {
        json_body = [for i, v in ["a", "b", "c"]: v if i < 2] # ["a", "b"]
      }
    }

    endpoint "/for5" {
      response {
        json_body = {for name in json_decode(request.headers.y): "${request.headers.z}-${name}" => request.headers[name]}
      }
    }

    # currently error: Unsupported attribute; This object does not have an attribute named "notthere".
    endpoint "/for6" {
      response {
        json_body = {for k, v in {}.notthere: k => v}
      }
    }
    endpoint "/for7" {
      response {
        json_body = {for k in ["a", "b"]: {}.notthere[k] => k}
      }
    }
    endpoint "/for8" {
      response {
        json_body = {for k in ["a", "b"]: k => {}.notthere[k]}
      }
    }
  }
}
