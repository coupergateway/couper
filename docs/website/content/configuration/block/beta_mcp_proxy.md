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
