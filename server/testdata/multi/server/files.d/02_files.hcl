server {
  files {
    document_root = "./"
    base_path = "/app"
  }

  files "another" {
    base_path = "/xxx"
    document_root = "./"
  }
}
