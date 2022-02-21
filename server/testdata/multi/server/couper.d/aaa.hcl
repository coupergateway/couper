server {
  files {
    document_root = "./www"
    custom_log_fields = {
      from = "final"
    }
  }

  endpoint "/free" {
    response {
      status = 403
    }
  }
}
