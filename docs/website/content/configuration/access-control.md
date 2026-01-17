---
title: 'Access Control'
description: 'List of configurable access control layer blocks.'
---

# Access Control

The configuration of access controls is twofold in Couper: You define the particular
type such as `jwt` or `basic_auth` in [`definitions`](/configuration/block/definitions), each with a distinct label
(must not be one of the reserved names: `granted_permissions`, `required_permission`).

Anywhere in the [`server` block](/configuration/block/server) those labels can be used in the `access_control`
list to protect the respective block.

> 📝 Access permissions are inherited by nested blocks.

You can also disable access control for blocks with `disable_access_control`: With `disable_access_control = ["bar"]`,
the `access_control` named `bar` will be disabled for the corresponding block context.

All access controls have an option to handle [related errors](/configuration/error-handling#access-control-error_handler).

### Blocks

* [`basic_auth`](/configuration/block/basic_auth)
* [`beta_oauth2`](/configuration/block/beta_oauth2)
* [`beta_rate_limiter`](/configuration/block/rate_limiter)
* [`jwt`](/configuration/block/jwt)
* [`oidc`](/configuration/block/oidc)
* [`saml`](/configuration/block/saml)
