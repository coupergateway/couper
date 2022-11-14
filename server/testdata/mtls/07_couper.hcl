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

    client_certificate "one" {
      leaf_certificate = <<-EOC
{{ .client1_Leaf }}
EOC
    }
#
    client_certificate "two" {
      leaf_certificate = <<-EOC
{{ .client2_Leaf }}
EOC
      ca_certificate = <<-EOC
{{ .client2_CA }}
EOC
    }

    client_certificate "three" {
      ca_certificate = <<-EOC
{{ .client3_CA }}
EOC
    }
  }
}
