server "hcl" {
  api {
    endpoint "/" {
      proxy {
        backend = "a"
      }
    }
    endpoint "/expired" {
      proxy {
        backend "b" {
          origin = "https://expired.badssl.com"
          path = "/"
        }
      }
    }
  }
}

definitions {
  backend "a" {
    origin = "https://blackhole.webpagetest.org"
    timeout = "2s"
  }

  backend "b" {
    origin = "http://1.2.3.4"
    disable_certificate_validation = true
  }

  basic_auth "parse-only" {}
}

settings {
  default_port = 8090
  no_proxy_from_env = true
}
