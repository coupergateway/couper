server "cty.NilVal" {
  endpoint "/1stchild" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR
        Z-Value = "y"
      }
    }
  }

  endpoint "/2ndchild/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR.not_there
        Z-Value = "y"
      }
    }
  }

  endpoint "/child-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = env.COUPER_TEST_BACKEND_ADDR.one.two
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.JSON.list[1]
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx-splat" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.JSON.list[*]
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.JSON.list[21]
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.JSON.list[21][12]
        Z-Value = "y"
      }
    }
  }

  endpoint "/list-idx-key-chain/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = backend_responses.default.json_body.JSON.list[21].obj[1]
        Z-Value = "y"
      }
    }
  }

  endpoint "/root/no" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = no-root
        Z-Value = "y"
      }
    }
  }

  endpoint "/tpl" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = "${env.COUPER_TEST_BACKEND_ADDR}mytext"
        Z-Value = "y"
      }
    }
  }

  endpoint "/for" {
    proxy {
      url = "${env.COUPER_TEST_BACKEND_ADDR}/anything"
    }
    response {
      headers = {
        X-Value = [for i, v in backend_responses.default.json_body.JSON.list: v if i < 1]
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/false" {
    response {
      headers = {
        X-Value = request.form_body.state != null ? request.form_body.state : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/true" {
    response {
      headers = {
        X-Value = true ? request.form_body.state : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested" {
    response {
      headers = {
        X-Value = request.form_body.state != (2 + 2 == 4 ? "value" : null) ? "x" : request.form_body.state
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested/true" {
    response {
      headers = {
        X-Value = request.form_body.state == null ? 1 + 1 == 2 ? "x" : null : request.form_body.state
        Z-Value = "y"
      }
    }
  }

  endpoint "/conditional/nested/false" {
    response {
      headers = {
        X-Value = request.form_body.state != null ? null : 1+1 == 3 ? null : "x"
        Z-Value = "y"
      }
    }
  }

  endpoint "/functions/arg-items" {
    response {
      headers = {
        X-Value = json_encode(merge({ "obj": {"key": "val"} }, {
          foo = "bar"
          xxxx = request.context.granted_permissions
        }))
        Z-Value = "y"
      }
    }
  }

  endpoint "/functions/tuple-expr" {
    response {
      headers = {
        X-Value = json_encode({ "array": merge(["a"], ["b"]) })
        Z-Value = "y"
      }
    }
  }

  endpoint "/rte1" {
    response {
      # *hclsyntax.RelativeTraversalExpr
      headers = {
        X-Value = [0, 2, 4][1]
        Z-Value = "y"
      }
    }
  }

  endpoint "/rte2" {
    response {
      # *hclsyntax.RelativeTraversalExpr
      headers = {
        X-Value = {a = 2, b = 4}["a"]
        Z-Value = "y"
      }
    }
  }

  endpoint "/ie1" {
    response {
      # *hclsyntax.IndexExpr
      headers = {
        X-Value = [0, 2, 4][2 - 1]
        Z-Value = "y"
      }
    }
  }

  endpoint "/ie2" {
    response {
      # *hclsyntax.IndexExpr
      headers = {
        X-Value = {"/ie1" = 1, "/ie2" = 2}[request.path]
        Z-Value = "y"
      }
    }
  }

  endpoint "/uoe1" {
    response {
      # *hclsyntax.UnaryOpExpr
      headers = {
        X-Value = -2
        Z-Value = "y"
      }
    }
  }

  endpoint "/uoe2" {
    response {
      # *hclsyntax.UnaryOpExpr
      headers = {
        X-Value = json_encode(!false)
        Z-Value = "y"
      }
    }
  }

  endpoint "/bad/dereference/string" {
    response {
      headers = {
        X-Value = request.query.foo[0].ooops
        Z-Value = "y"
      }
    }
  }

  endpoint "/bad/dereference/array" {
    response {
      headers = {
        X-Value = request.query.foo.ooops
        Z-Value = "y"
      }
    }
  }


  endpoint "/conditional/null" {
    response {
      status = null ? 204 : 400
    }
  }

  endpoint "/conditional/string" {
    response {
      status = "foo" ? 204 : 400
    }
  }

  endpoint "/conditional/number" {
    response {
      status = 2 ? 204 : 400
    }
  }

  endpoint "/conditional/tuple" {
    response {
      status = [] ? 204 : 400
    }
  }

  endpoint "/conditional/object" {
    response {
      status = {} ? 204 : 400
    }
  }

  endpoint "/conditional/string/expr" {
    response {
      status = {"a": "foo"}.a ? 204 : 400
    }
  }

  endpoint "/conditional/number/expr" {
    response {
      status = {"a": 2}.a ? 204 : 400
    }
  }
}

settings {
  no_proxy_from_env = true
}
