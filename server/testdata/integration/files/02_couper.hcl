server "multi-files-host1" {
  hosts = ["*", "couper.io:9898"]
  error_file = "./../server_error.html"

  files {
    base_path = "/a"
    document_root = "./htdocs_a"
  }
}

server "multi-files-host2" {
  hosts = ["example.com:9898"]
  error_file = "./../server_error.html"

  base_path = "/b"
  files {
    document_root = "./htdocs_b"
  }
}
