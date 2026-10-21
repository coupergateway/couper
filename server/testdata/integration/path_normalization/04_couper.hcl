# Two file mounts with overlapping names. The protected mount must win for
# every variant of /private/secret.txt.
server "files" {
  files {
    base_path      = "/private"
    document_root  = "./htdocs_private"
    access_control = ["creds"]
  }

  files {
    document_root = "./htdocs_public"
  }
}

definitions {
  basic_auth "creds" {
    user     = "alice"
    password = "secret"
  }
}
