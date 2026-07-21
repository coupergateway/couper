server {
  api {
    endpoint "/mcp" {
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
