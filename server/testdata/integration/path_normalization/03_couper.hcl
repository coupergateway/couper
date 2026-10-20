# Patterns with a fixed number of segments. "." and ".." are valid path
# parameter values, so a variant with a matching segment count reaches
# the unprotected endpoint.
server "gateway" {
  hosts = ["*:8080"]

  endpoint "/admin/{file}" {
    access_control = ["creds"]
    proxy {
      backend = "origin"
    }
  }

  endpoint "/{area}/{file}" {
    proxy {
      backend = "origin"
    }
  }

  endpoint "/{a}/{b}/{c}/{d}" {
    proxy {
      backend = "origin"
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

  endpoint "/**" {
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
}
