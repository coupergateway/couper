server {
  hosts = ["*:4443"]

  endpoint "/" {
    response {}
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
