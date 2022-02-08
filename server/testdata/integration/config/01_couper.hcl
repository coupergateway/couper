server "hcl" {
  api {
    endpoint "/" {
      proxy {
        backend = "a"
      }
    }
    endpoint "/expired" {
      proxy {
        backend = "b"
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
    disable_certificate_validation = true
    origin = "https://expired.badssl.com"
    path = "/"
  }

  basic_auth "parse-only" {}
}

settings {
  default_port = 8090
  no_proxy_from_env = true
}
