---
title: 'Basic Auth'
slug: 'basic_auth'
---

# Basic Auth

| Block name   | Context                                               | Label    |
|:-------------|:------------------------------------------------------|:---------|
| `basic_auth` | [Definitions Block](/configuration/block/definitions) | required |

The `basic_auth` block lets you configure HTTP basic auth for your gateway. Like all
[access control](/configuration/access-control) types, the `basic_auth` block is defined in the
[`definitions` block](/configuration/block/definitions) and can be referenced in all configuration
blocks by its required _label_.

Basic Auth is intended for simple access control situations where a static list
of users is sufficient. This could be to protect a staging environment, or to
expose a dedicated API to a single internal client, such as a neighboring
microservice.

The `user` is accessible via `request.context.<label>.user` variable for successfully authenticated requests.

If both `user`/`password` and `htpasswd_file` are configured, the incoming
credentials from the `Authorization` request HTTP header field are checked against
`user`/`password` if the user matches, and against the data in the file referenced
by `htpasswd_file` otherwise.

## Example

### Using inline credentials

```hcl
server {
  api {
    endpoint "/private" {
      access_control = ["myauth"]
      proxy {
        backend = "my_backend"
      }
    }
  }
}

definitions {
  basic_auth "myauth" {
    user     = "john"
    password = "s3cr3t"
  }
}
```

### Using an htpasswd file

```hcl
definitions {
  basic_auth "myauth" {
    htpasswd_file = "htpasswd"
  }
}
```

The htpasswd file uses [Apache's htpasswd](https://httpd.apache.org/docs/current/programs/htpasswd.html) format:

```
john:$2y$05$/uonQYUtwm...
jane:$argon2id$v=19$m=65536,t=3,p=2$salt$hash
```

### Attribute `htpasswd_file`

The file is loaded once at startup. Restart Couper after you have changed it.

Couper supports the following password hash algorithms:

| Algorithm  | htpasswd prefix | Recommended |
|:-----------|:----------------|:------------|
| `argon2id` | `$argon2id$`    | yes         |
| `argon2i`  | `$argon2i$`     |             |
| `bcrypt`   | `$2y$`          |             |
| `md5`      | `$apr1$`        |             |

### Choosing Argon2 parameters for security and performance

When generating your own password hashes, **`argon2id` is the recommended choice** as it provides a balanced approach to resisting both side-channel and GPU-based attacks (see [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)).

The argon2 hash encodes the parameters used to derive it: `m` (memory in KiB), `t` (iterations) and `p` (parallelism). Couper re-runs the key derivation with these parameters on every authenticated request, so the choice has both security and runtime cost implications.

OWASP currently recommends for `argon2id`: `m=19456` (≈19 MiB), `t=2`, `p=1`.

_Memory cost is per request_. Couper allocates `m` KiB on every basic auth verification, which is why parameter choice matters for the gateway's resident memory under load. Couper caps the parameters encoded in the htpasswd file at twice the highest OWASP-recommended values and refuses to start otherwise:

| Parameter   | Cap         | OWASP highest |
|:------------|:------------|:--------------|
| `m` (KiB)   | `94208`     | `47104`       |
| `t`         | `10`        | `5`           |
| `p`         | `2`         | `1`           |

To bound amplification under retry storms, Couper collapses concurrent identical verifications into a single argon2 evaluation and caches both positive and negative results for five minutes per `(user, password)` pair. A single unique attempt still pays the full derivation cost — see "Pair with a rate limiter" below.

### Pair with a rate limiter

Argon2 is intentionally expensive. Even with the parameter cap and the in-process result cache, an attacker who cycles through unique wrong passwords pays no cache cost and forces Couper through one full derivation per attempt. Place a [`beta_rate_limiter`](/configuration/block/rate_limiter) access control _before_ the basic auth in the endpoint's `access_control` list so abusive callers are rejected before any argon2 work runs:

```hcl
server {
  api {
    endpoint "/private" {
      access_control = ["ip_rate", "myauth"]
      proxy {
        backend = "my_backend"
      }
    }
  }
}

definitions {
  beta_rate_limiter "ip_rate" {
    period        = "60s"
    per_period    = 10
    period_window = "sliding"
    key           = request.remote_ip
  }

  basic_auth "myauth" {
    htpasswd_file = "htpasswd"
  }
}
```

Access controls run in the order listed: the rate limiter rejects the request first, so basic auth is invoked only for callers within the budget.


{{< attributes >}}
[
  {
    "default": "",
    "description": "Log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks.",
    "name": "custom_log_fields",
    "type": "object"
  },
  {
    "default": "",
    "description": "The htpasswd file.",
    "name": "htpasswd_file",
    "type": "string"
  },
  {
    "default": "",
    "description": "The corresponding password.",
    "name": "password",
    "type": "string"
  },
  {
    "default": "",
    "description": "The realm to be sent in a WWW-Authenticate response HTTP header field.",
    "name": "realm",
    "type": "string"
  },
  {
    "default": "",
    "description": "The user name.",
    "name": "user",
    "type": "string"
  }
]
{{< /attributes >}}

{{< blocks >}}
[
  {
    "description": "Configures an [error handler](/configuration/block/error_handler) (zero or more).",
    "name": "error_handler"
  }
]
{{< /blocks >}}
