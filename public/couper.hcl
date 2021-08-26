server "couper" {
  files {
    document_root = env.DOC_DIR
  }
}

defaults {
  environment_variables = {
    DOC_DIR = "./"
  }
}
