server {
  api {
  }
  spa {
    bootstrap_file = "./couper_03.hcl"
    paths = ["/**"]
  }
  files {
    document_root = "./"
  }
}
