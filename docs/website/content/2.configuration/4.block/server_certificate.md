---
title: 'Server Certificate'
description: 'The `server_certificate` block is part of its parent `tls` block. Enables TLS configuration.'
draft: false
---

# Server Certificate

| Block name   | Context                                              | Label    |
|:-------------|:-----------------------------------------------------|:---------|
| `server_certificate` | [tls Block](/configuration/block/server_tls) | optional |

A configured `server_certificate` pub/key pair will be loaded once at startup and served on the related `server` port configured with the `hosts` attribute.

## Example

```hcl
server_certificate "api.example.com" { #optional label
  public_key = "base64_DER" # kube secret ...
  # OR
  public_key_file = "couperServer.crt" # PEM

  private_key = "base64_DER"
  # OR
  private_key_file = "couperServer.key" # PEM
}
```

::attributes
---
values: [
  {
    "default": "",
    "description": "Private part of the certificate in DER or PEM format. Mutually exclusive with `private_key_file`.",
    "name": "private_key",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the private part of the certificate file in DER or PEM format. Mutually exclusive with `private_key`.",
    "name": "private_key_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "Public part of the certificate in DER or PEM format. Mutually exclusive with `public_key_file`.",
    "name": "public_key",
    "type": "string"
  },
  {
    "default": "",
    "description": "Reference to a file containing the public part of the certificate file in DER or PEM format. Mutually exclusive with `public_key`.",
    "name": "public_key_file",
    "type": "string"
  }
]

---
::

::blocks
---
values: null

---
::
