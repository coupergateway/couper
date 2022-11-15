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
      leaf_certificate = <<-EOC
{{ .clientLeaf }}
EOC
    }
  }
}
