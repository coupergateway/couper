server "protected" {
  hosts = ["*:4443"]

  api {
    endpoint "/protected" {
      access_control = ["authz"]

      response {
        status = 204
      }
    }
  }

  tls {
    server_certificate {
      public_key = <<-EOC
{{ .publicKey }}
EOC
      private_key = <<-EOC
{{ .privateKey }}
EOC
    }

    client_certificate {
      ca_certificate = <<-EOC
{{ .clientCA }}
EOC
    }
  }
}

definitions {
  beta_external_authz "authz" {
    url         = "{{.origin}}/check"
    include_tls = true
  }
}
