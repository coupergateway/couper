server {
  hosts = ["*:8080"]

  api {
    endpoint "/anything" {
      proxy {
        backend = "oauth_rs_cc"
      }
    }
  }
}

definitions {
  backend "oauth_rs_cc" {
    origin = "https://httpbin.org"

    oauth2 {
      grant_type = "client_credentials"
      client_id = "clid1"
      client_secret = "cls1"
      token_endpoint = "http://localhost:8081/token"
      backend = "oauth_as1"
    }
  }

  backend "oauth_as1" {
    origin = "http://localhost:8081"

    oauth2 {
      grant_type = "client_credentials"
      client_id = "csid2"
      client_secret = "cls2"
      token_endpoint = "http://localhost:8082/token"
      backend = "oauth_as2"
    }
  }
  backend "oauth_as2" {
    origin = "http://localhost:8082"
  }
}

server "as1" {
  hosts = ["*:8081"]

  api {
    endpoint "/token" {
      response {
        json_body = {
          access_token = "foo-${unixtime()}"
          expires_in = 60
        }
      }
    }
  }
}

server "as2" {
  hosts = ["*:8082"]

  api {
    endpoint "/token" {
      response {
        json_body = {
          access_token = "lkjh"
          expires_in = "1h"
        }
      }
    }
  }
}
