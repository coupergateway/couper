server {
  api {
    endpoint "/mcp" {
      beta_mcp_proxy {
        backend = "mcp_server"
        allowed_tools = ["get_weather", "read_*"]
        blocked_tools = ["read_secret"]
      }
    }

    endpoint "/mcp-block-only" {
      beta_mcp_proxy {
        backend = "mcp_server"
        blocked_tools = ["delete_*", "exec_*"]
      }
    }

    endpoint "/mcp-passthrough" {
      beta_mcp_proxy {
        backend = "mcp_server"
      }
    }
  }
}

definitions {
  backend "mcp_server" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
  }
}
