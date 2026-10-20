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

server "origin" {
  hosts = ["*:9898"]

  endpoint "/admin/panel.txt" {
    response {
      body = "TOP SECRET admin panel"
    }
  }

  endpoint "/files/**" {
    response {
      body = "origin saw ${request.url}"
    }
  }
}

definitions {
  basic_auth "creds" {
    user     = "alice"
    password = "secret"
  }

  backend "origin" {
    origin = "http://localhost:9898"
  }

  backend "origin_files" {
    origin = "http://localhost:9898"
    path   = "/files/**"
  }

  backend "origin_prefixed" {
    origin      = "http://localhost:9898"
    path_prefix = "/files"
  }
}
