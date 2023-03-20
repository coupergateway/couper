server {
  hosts = ["*:4443"]

  endpoint "/" {
    response {
      headers = {
        location = request.url
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
  }
}
