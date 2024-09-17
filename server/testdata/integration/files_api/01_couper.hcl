server "files" {
  files "htdocs" {
    base_path = "/"
    document_root = "./"
  }
  files "assets" {
    base_path = "/assets"
    document_root = "./"
  }
  api {
  }
}
