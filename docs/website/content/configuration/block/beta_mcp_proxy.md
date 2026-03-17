---
title: 'MCP Proxy (Beta)'
slug: 'mcp_proxy'
---

# MCP Proxy (beta)

The `beta_mcp_proxy` block creates a proxy for [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) servers with tool-level filtering. It intercepts JSON-RPC `tools/list` responses to filter exposed tools and blocks unauthorized `tools/call` requests before they reach the backend.

| Block name       | Context                                         | Label    |
|:-----------------|:------------------------------------------------|:---------|
| `beta_mcp_proxy` | [Endpoint Block](/configuration/block/endpoint) | Optional |

## Tool Filtering

Use `allowed_tools` and `blocked_tools` attributes with glob patterns to control which MCP tools are exposed:

- `allowed_tools` set: only matching tools are exposed (allowlist)
- `blocked_tools` set: matching tools are hidden (denylist)
- Both set: tool must match allowed AND not match blocked
- Neither set: all tools pass through (transparent proxy)

Both attributes support runtime expressions and can reference JWT claims (e.g. `ctx.my_jwt.allowed_tools`).

Blocked `tools/call` requests return a JSON-RPC error (`-32601 Method not found`) without forwarding to the backend.

### Example: Static tool filtering

```hcl
server {
  api {
    endpoint "/mcp" {
      beta_mcp_proxy {
        backend = "mcp-server"
        allowed_tools = ["get_weather", "search_*", "read_*"]
      }
    }
  }
}

definitions {
  backend "mcp-server" {
    origin = "http://localhost:3001"
  }
}
```

### Example: Block dangerous tools (denylist)

```hcl
beta_mcp_proxy {
  backend = "mcp-server"
  blocked_tools = ["delete_*", "drop_*", "exec_*", "rm_*"]
}
```

### Example: Per-user tool access via JWT claims

```hcl
server {
  api {
    endpoint "/mcp" {
      access_control = ["mcp_jwt"]

      beta_mcp_proxy {
        backend = "mcp-server"
        allowed_tools = ctx.mcp_jwt.allowed_tools
      }
    }
  }
}

definitions {
  jwt "mcp_jwt" {
    signature_algorithm = "RS256"
    jwks_url = "https://auth.example.com/.well-known/jwks.json"
  }

  backend "mcp-server" {
    origin = "http://localhost:3001"
  }
}
```

### Example: Nested claims with flatten()

```hcl
beta_mcp_proxy {
  backend = "mcp-server"
  # JWT payload: {"mcp_permissions": {"server1": ["tool_a"], "server2": ["tool_b"]}}
  allowed_tools = flatten(values(ctx.tenant_jwt.mcp_permissions))
  blocked_tools = ["admin_*", "debug_*"]
}
```

## OAuth Authentication

When the upstream MCP server requires OAuth authentication (e.g. [MCP OAuth per RFC 9728](https://datatracker.ietf.org/doc/html/rfc9728)), Couper automatically handles the OAuth discovery and token flow. The following endpoints are registered alongside the MCP proxy endpoint:

| Path | Purpose |
|:-----|:--------|
| `/.well-known/oauth-protected-resource` | Serves rewritten protected resource metadata |
| `/.well-known/oauth-authorization-server` | Serves rewritten authorization server metadata |
| `/token` | Proxies token requests to upstream |
| `/register` | Proxies dynamic client registration to upstream |

### How it works

```
MCP Client ──► Couper (proxy) ──► Upstream MCP Server
    │                                      │
    │  1. POST /mcp → 401                  │
    │  2. GET /.well-known/oauth-protected-resource
    │     ← resource rewritten to proxy URL │
    │  3. GET /.well-known/oauth-authorization-server
    │     ← token/register endpoints        │
    │       rewritten to proxy URLs         │
    │     ← authorization_endpoint stays    │
    │       pointing at upstream (browser)  │
    │  4. POST /register → forwarded        │
    │  5. Browser → upstream /authorize     │
    │  6. POST /token → resource param      │
    │     rewritten to upstream origin      │
    │  7. POST /mcp + Bearer token → OK     │
```

The key challenge with proxying MCP OAuth is that tokens are bound to the `resource` value used during issuance. Couper solves this by:

1. **Advertising the proxy URL** as the `resource` in discovery metadata — so MCP clients accept the proxy as the resource server
2. **Rewriting the `resource` parameter** in `/token` and `/register` requests back to the upstream origin — so the upstream issues tokens bound to its own origin
3. **Keeping `authorization_endpoint`** pointing directly at the upstream — browser redirects cannot be proxied through the API gateway

This happens transparently. No additional configuration is needed beyond the `beta_mcp_proxy` block.

### Forwarding the Authorization header

Couper strips `Authorization` headers from proxy requests by default. To forward Bearer tokens to the upstream MCP server, use `set_request_headers`:

```hcl
beta_mcp_proxy {
  backend = "mcp-server"
  set_request_headers = {
    authorization = request.headers.authorization
  }
}
```

This works for any authentication scheme (Bearer, Basic, etc.).

### Example: OAuth-protected MCP server

```hcl
server {
  api {
    endpoint "/mcp" {
      beta_mcp_proxy {
        backend = "mcp-server"
        allowed_tools = ["get_weather", "search_*"]
        set_request_headers = {
          authorization = request.headers.authorization
        }
      }
    }
  }
}

definitions {
  backend "mcp-server" {
    origin = "https://mcp.example.com"
  }
}
```

The OAuth discovery and token endpoints are registered automatically. MCP clients connecting to `http://your-gateway/mcp` will complete the OAuth flow transparently through the proxy.

### Non-OAuth MCP servers

If the upstream MCP server does not use OAuth, the auto-registered OAuth endpoints are harmless — they return an error only if explicitly called. Basic auth, API keys, and other authentication methods work normally via `set_request_headers`.

{{< attributes >}}
[
  {
    "default": "",
    "description": "Key/value pairs to add form parameters to the upstream request body.",
    "name": "add_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add query parameters to the upstream request URL.",
    "name": "add_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as request headers in the upstream request.",
    "name": "add_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to add as response headers in the client response.",
    "name": "add_response_headers",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "List of tool name patterns (glob) to allow. If set, only matching tools are exposed.",
    "name": "allowed_tools",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "References a [backend](/configuration/block/backend) in [definitions](/configuration/block/definitions) for the MCP proxy request. Mutually exclusive with `backend` block.",
    "name": "backend",
    "type": "string"
  },
  {
    "default": "[]",
    "description": "List of tool name patterns (glob) to block. Matching tools are hidden.",
    "name": "blocked_tools",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "List of names to remove form parameters from the upstream request body.",
    "name": "remove_form_params",
    "type": "object"
  },
  {
    "default": "[]",
    "description": "List of names to remove query parameters from the upstream request URL.",
    "name": "remove_query_params",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the upstream request.",
    "name": "remove_request_headers",
    "type": "tuple (string)"
  },
  {
    "default": "[]",
    "description": "List of names to remove headers from the client response.",
    "name": "remove_response_headers",
    "type": "tuple (string)"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_form_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set query parameters in the upstream request URL.",
    "name": "set_query_params",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as request headers in the upstream request.",
    "name": "set_request_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "Key/value pairs to set as response headers in the client response.",
    "name": "set_response_headers",
    "type": "object"
  },
  {
    "default": "",
    "description": "URL of the resource to request. May be relative to an origin specified in a referenced or nested `backend` block.",
    "name": "url",
    "type": "string"
  }
]
{{< /attributes >}}

{{< blocks >}}
[
  {
    "description": "Configures a [backend](/configuration/block/backend) for the MCP proxy request (zero or one). Mutually exclusive with `backend` attribute.",
    "name": "backend"
  }
]
{{< /blocks >}}
