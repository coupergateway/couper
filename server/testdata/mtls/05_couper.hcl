server {
  hosts = ["*:4443"]

  endpoint "/" {
    response {}
  }

  endpoint "/inception" {
    proxy {
      backend = "secured"
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
      leaf_certificate = <<-EOC
{{ .clientLeaf }}
EOC
    }
  }
}

definitions {
  backend "secured" {
    origin = "https://localhost:4443"
    path = "/"

    tls {
      server_ca_certificate = <<-EOC
{{ .rootCA }}
EOC
      client_certificate = <<-EOC
{{ .clientLeaf }}
EOC
      client_private_key = <<-EOC
{{ .clientKey }}
EOC
    }
  }
}
