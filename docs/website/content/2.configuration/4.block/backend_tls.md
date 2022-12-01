---
title: 'TLS (Backend)'
description: 'TLS settings for the related backend.'
draft: false
---

# TLS (Backend)

| Block name   | Context                                       | Label    |
|:-------------|:----------------------------------------------|:---------|
| `tls`        | [Backend Block](/configuration/block/backend) | no       |

Couper has a command-line argument to add a `ca-file` to the backend CA-Pool for all backends.
However, this `tls` block allows a more specific pool configuration per backend if the `server_ca_certificate` or
`server_ca_certificate_file` is provided.

### mTLS

Additionally the `client_certificate`(or `client_certificate_file`) and `client_private_key` (or `client_private_key_file`)
attributes allow the backend to present certificate and key during a TLS handshake to an origin which requires them due to an mTLS setup.

#### Example

```hcl
backend "secured" {
    origin = "https://localhost"

    tls {
      server_ca_certificate_file = "rootCA.crt"
      # optional
      client_certificate_file = "client.crt"
      client_private_key_file = "client.key"
    }
  }
```

::attributes
---
values: [
  {
    "default": "",
    "description": "Public part of the client certificate in DER or PEM format. Mutually exclusive with `client_certificate_file`.",
    "name": "client_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the public part of the client certificate file in DER or PEM format. Mutually exclusive with `client_certificate`.",
    "name": "client_certificate_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Private part of the client certificate in DER or PEM format. Required to complete an mTLS handshake. Mutually exclusive with `client_private_key_file`.",
    "name": "client_private_key",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the private part of the client certificate file in DER or PEM format. Required to complete an mTLS handshake. Mutually exclusive with `client_private_key`.",
    "name": "client_private_key_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the certificate authority in DER or PEM format. Mutually exclusive with `server_ca_certificate_file`.",
    "name": "server_ca_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the public part of the certificate authority file in DER or PEM format. Mutually exclusive with `server_ca_certificate`.",
    "name": "server_ca_certificate_file",
    "type": "string"
  }
]

---
::
