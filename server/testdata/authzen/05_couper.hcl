server "protected" {
  hosts = ["*:8080"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        headers = {
          x-authz-proto = request.context.authz.proto
        }
      }
    }
  }
}

definitions {
  beta_authzen "authz" {
    backend {
      origin = "{{.origin}}"
      http2  = true

      tls {
        server_ca_certificate = <<-EOC
{{ .ca }}
EOC
      }
    }
  }
}
