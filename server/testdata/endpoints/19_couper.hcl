server { # sequences
  hosts = ["*:8080"]

  api {
    endpoint "/" {
      request "r1" {
        url = "/res1"
        backend = "src"
      }

      request "r2" {
        url = "/res2"
        backend = "src"
      }

      request { # default
        url = "/"
        backend = "dest"
        json_body = {
          data = [
            backend_responses.r1.json_body,
            backend_responses.r2.json_body
          ]
        }
      }
    }
  }
}

definitions {
  backend "src" {
    origin = "http://localhost:8081"
  }

  backend "dest" {
    origin = "http://localhost:8082"
  }
}

server "src" {
  hosts = ["*:8081"]

  api {
    endpoint "/res1" {
      response {
        json_body = {
          features = 1
        }
      }
    }

    endpoint "/res2" {
      response {
        json_body = {
          features = 2
        }
      }
    }
  }
}

server "dest" {
  hosts = ["*:8082"]

  api {
    endpoint "/" {
      response {
        json_body = request.json_body
      }
    }
  }
}
