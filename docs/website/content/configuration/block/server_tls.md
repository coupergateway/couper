---
title: 'TLS (Server)'
slug: 'server_tls'
description: 'TLS settings for the related server.'
draft: false
---

# TLS (Server)

| Block name   | Context                                     | Label    |
|:-------------|:--------------------------------------------|:---------|
| `tls`        | [Server Block](/configuration/block/server) | no       |

The `tls` block allows to configure one or more server certificates in the first place.
The certificates will be served on all ports within the `hosts` list. Enabling `tls` also enables the upgrade option to the `HTTP2` protocol.

 > The simplest configuration is an empty `tls {}` block which will serve a self signed certificate
for local development.

Multiple [`server_certificate`](server_certificate) or [`client_certificate`](client_certificate) blocks are allowed.

## mTLS

Once a [`client_certificate`](client_certificate) block is defined the server automatically requests and verify a certificate from the client.

## Example

```hcl
server "couper" {
  hosts = ["*:443"]

  tls {
    server_certificate "api.example.com" {
      public_key_file = "couperServer.crt" # PEM
      private_key_file = "couperServer.key" # PEM
    }

    # mTLS

    client_certificate "IOT" {
      ca_certificate_file = "couperIntermediate.crt" # PEM
      # OR(AND!)
      leaf_certificate_file = "couperClient.crt" # PEM
    }
  }
```

{{< blocks >}}
[
  {
    "description": "Configures a [client certificate](/configuration/block/client_certificate) (zero or more).",
    "name": "client_certificate"
  },
  {
    "description": "Configures a [server certificate](/configuration/block/server_certificate) (zero or more).",
    "name": "server_certificate"
  }
]
{{< /blocks >}}
