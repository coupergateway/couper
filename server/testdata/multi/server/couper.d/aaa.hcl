server {
  files {
    document_root = "./web"
    custom_log_fields = {
      from = "final"
    }
  }

  endpoint "/free" {
    response {
      status = 401
    }
  }
}
