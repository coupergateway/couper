# The url attribute builds the upstream URL from request data. A traversal
# sequence in that data must not reach the origin.
server "gateway" {
  hosts = ["*:8080"]

  endpoint "/u" {
    proxy {
      url = "${env.COUPER_TEST_RAW_BACKEND_ADDR}/files/${request.query.f[0]}"
    }
  }

  endpoint "/r" {
    request "r" {
      url = "${env.COUPER_TEST_RAW_BACKEND_ADDR}/files/${request.query.f[0]}"
    }

    response {
      body = backend_responses.r.body
    }
  }
}

# Request data at the start of the url value. An opaque value such as
# "http:http://host/.." has no path to check and must be rejected.
server "opaque" {
  hosts = ["*:9897"]

  endpoint "/ou" {
    proxy {
      backend = "origin"
      url     = request.query.u[0]
    }
  }

  endpoint "/or" {
    request "r" {
      backend = "origin"
      url     = request.query.u[0]
    }

    response {
      body = backend_responses.r.body
    }
  }
}

definitions {
  backend "origin" {
    origin = env.COUPER_TEST_RAW_BACKEND_ADDR
  }
}
