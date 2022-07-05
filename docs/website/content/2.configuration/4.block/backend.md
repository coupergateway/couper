---
title: 'Backend'
description: 'The backend defines the connection pool with given origin for outgoing connections.'
draft: false
---

# Backend

The `backend` block defines the connection to a local/remote backend service.

&#9888; Backends can be defined in the [Definitions Block](#definitions-block) and referenced by _label_.

| Block name | Context                                                                                                                                                                                                                                   | Label                                                                     | Nested block(s)                                                                                  |
|:-----------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------------------------------------------|:-------------------------------------------------------------------------------------------------|
| `backend`  | [Definitions Block](#definitions-block), [Proxy Block](#proxy-block), [Request Block](#request-block), [OAuth2 CC Block](#oauth2-block), [JWT Block](#jwt-block), [OAuth2 AC Block (beta)](#beta-oauth2-block), [OIDC Block](#oidc-block) | &#9888; required, when defined in [Definitions Block](#definitions-block) | [OpenAPI Block](#openapi-block), [OAuth2 CC Block](#oauth2-block), [Health Block](#health-block) |

::attributes
::

::duration
