---
title: 'Client Certificate'
description: 'The `client_certificate` block is part of its parent `tls` block. Enables mTLS configuration.'
draft: false
---

# Client Certificate

| Block name   | Context                                              | Label    |
|:-------------|:-----------------------------------------------------|:---------|
| `client_certificate` | [tls Block](/configuration/block/server_tls) | optional |

Define an optional `client_certificate` block with its optional _label_ to enable **mTLS**.

> **mTLS:** stands for mutual TLS and will extend the normal handshake process with an additional request (client must present the certificate) and verification for the configured client certificate (CA).

Configuring a `ca_certificate` is the standard way to specify a client certificate. But you can also provide the `leaf_certificate`
which effectively is the client certificate. The server will verify the given client certificate byte by byte against its own leaf certificate.
A combination of `ca_certificate`(or `ca_certificate_file`) or/and `leaf_certificate`(or `leaf_certificate_file`) is valid.
This covers the use-case where the CA has signed multiple client certificates and you want to limit the access to specific ones.

## Example

```hcl
client_certificate "IOT" {
  ca_certificate = "base64_der" # PEM or DER encoded
  # OR
  ca_certificate_file = "couperIntermediate.crt" # PEM

  # OR/AND
  # trusted client leaf cert

  leaf_certificate = "base64_der"
  # OR
  leaf_certificate_file = "couperClient.crt" # PEM
}
```

::attributes
---
values: [
  {
    "default": "",
    "description": "Public part of the certificate authority in DER or PEM format. Mutually exclusive with `ca_certificate_file`.",
    "name": "ca_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the public part of the certificate authority file in DER or PEM format. Mutually exclusive with `ca_certificate`.",
    "name": "ca_certificate_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the client certificate in DER or PEM format. Mutually exclusive with `leaf_certificate_file`.",
    "name": "leaf_certificate",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the public part of the client certificate file in DER or PEM format. Mutually exclusive with `leaf_certificate`.",
    "name": "leaf_certificate_file",
    "type": "string"
  }
]

---
::
