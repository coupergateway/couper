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
    origin = "{{ .expiredOrigin }}"
    path = "/anything"
    disable_certificate_validation = true
  }

  basic_auth "parse-only" {}
}

settings {
  default_port = 8090
  no_proxy_from_env = true
  ca_file = "{{ .caFile }}"
}
