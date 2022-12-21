# Token Introspection

The `introspection` block lets you configure OAuth2 token introspection for an encapsulating `jwt` block.

| Block name      | Context                               | Label    |
|:----------------|:--------------------------------------|:---------|
| `introspection` | [JWT Block](/configuration/block/jwt) | no label |

::attributes
---
values: [
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for introspection requests. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "",
    "description": "The authorization server's `introspection_endpoint`.",
    "name": "endpoint",
    "type": "string"
  },
  {
    "default": "",
    "description": "The time-to-live of a cached introspection response.",
    "name": "ttl",
    "type": "string"
  }
]

---
::

::blocks
---
values: [
  {
    "description": "Configures a [backend](/configuration/block/backend) for introspection requests (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  }
]

---
::
