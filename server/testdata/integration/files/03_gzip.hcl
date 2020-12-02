server "serve-gzip" {
  hosts = ["example.org:9898"]
  error_file = "./../server_error.html"

  files {
    document_root = "./htdocs_c_gzip"
  }
}
