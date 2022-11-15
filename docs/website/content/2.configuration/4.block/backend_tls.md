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
attributes allow the backend to present those ones during a TLS handshake to an origin which requires them due to an mTLS setup.

::attributes
---
values: [
  {
    "default": "",
    "description": "Public part of the client certificate in DER or PEM format.",
    "name": "client_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the client certificate file in DER or PEM format.",
    "name": "client_certificate_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Private part of the client certificate in DER or PEM format. Required to complete a mTLS handshake.",
    "name": "client_private_key",
    "type": "string"
  },
  {
    "default": "",
    "description": "Private part of the client certificate file in DER or PEM format. Required to complete a mTLS handshake.",
    "name": "client_private_key_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the certificate authority in DER or PEM format.",
    "name": "server_ca_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the certificate authority file in DER or PEM format.",
    "name": "server_ca_certificate_file",
    "type": "string"
  }
]

---
::
