server {
  api {
    base_path = "/cc-private"
    access_control = ["jott"]

    endpoint "/no-cc" {
      response {
        status = 204
      }
    }

    endpoint "/cc-public" {
      response {
        status = 204
        headers = {
          cache-control = "public"
        }
      }
    }
  }
  api {
    base_path = "/no-cc-private"
    access_control = ["jott-disable-cc-private"]

    endpoint "/no-cc" {
      response {
        status = 204
      }
    }

    endpoint "/cc-public" {
      response {
        status = 204
        headers = {
          cache-control = "public"
        }
      }
    }
  }
}

definitions {
  jwt "jott" {
    signature_algorithm = "HS256"
    key = "asdf"
  }

  jwt "jott-disable-cc-private" {
    signature_algorithm = "HS256"
    key = "asdf"
    disable_private_caching = true
  }
}
