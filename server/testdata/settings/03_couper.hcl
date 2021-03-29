server "log-format-common" {
  files {
    document_root = "./"
  }
}

settings {
  request_id_format = "uuid4"
}
