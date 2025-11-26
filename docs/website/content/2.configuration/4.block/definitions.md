# Definitions

Use the `definitions` block to define configurations you want to reuse.

&#9888; [access control](/configuration/access-control) is **always** defined in the `definitions` block.

| Block name    | Context | Label    |
|:--------------|:--------|:---------|
| `definitions` | -       | no label |

> Legacy configurations may still contain `beta_job` blocks; they remain supported as aliases for `job`.

::blocks
---
values: [
  {
    "description": "Configure a [backend](/configuration/block/backend) (zero or more).",
    "name": "backend"
  },
  {
    "description": "Configure a [BasicAuth access control](/configuration/block/basic_auth) (zero or more).",
    "name": "basic_auth"
  },
  {
    "description": "Configure a [job](/configuration/block/job) (zero or more).",
    "name": "job"
  },
  {
    "description": "Configure an [OAuth2 access control](/configuration/block/beta_oauth2) (zero or more).",
    "name": "beta_oauth2"
  },
  {
    "description": "Configure a [Rate limiter access control](/configuration/block/rate_limiter) (zero or more).",
    "name": "beta_rate_limiter"
  },
  {
    "description": "Configure a [JWT access control](/configuration/block/jwt) (zero or more).",
    "name": "jwt"
  },
  {
    "description": "Configure a [JWT signing profile](/configuration/block/jwt_signing_profile) (zero or more).",
    "name": "jwt_signing_profile"
  },
  {
    "description": "Configure an [OIDC access control](/configuration/block/oidc) (zero or more).",
    "name": "oidc"
  },
  {
    "description": "Configure a [proxy](/configuration/block/proxy) (zero or more).",
    "name": "proxy"
  },
  {
    "description": "Configure a [SAML access control](/configuration/block/saml) (zero or more).",
    "name": "saml"
  }
]

---
::
