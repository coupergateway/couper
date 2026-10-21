# Rewrites: path = "/files/**" and path_prefix = "/files" build the upstream
# path with path.Join. Before normalization a ".." in the request escaped
# the configured prefix and the origin served the protected resource.
server "gateway" {
  hosts = ["*:8080"]

  endpoint "/admin/**" {
    access_control = ["creds"]
    proxy {
      backend = "origin"
    }
  }

  endpoint "/{area}/**" {
    proxy {
      backend = "origin_files"
    }
  }
}

server "gateway_prefix" {
  hosts = ["*:9897"]

  endpoint "/admin/**" {
    access_control = ["creds"]
    proxy {
      backend = "origin"
    }
  }

  endpoint "/{area}/**" {
    proxy {
      backend = "origin_prefixed"
    }
  }
}

definitions {
  basic_auth "creds" {
    user     = "alice"
    password = "secret"
  }

  backend "origin" {
    origin = env.COUPER_TEST_RAW_BACKEND_ADDR
  }

  backend "origin_files" {
    origin = env.COUPER_TEST_RAW_BACKEND_ADDR
    path   = "/files/**"
  }

  backend "origin_prefixed" {
    origin      = env.COUPER_TEST_RAW_BACKEND_ADDR
    path_prefix = "/files"
  }
}
