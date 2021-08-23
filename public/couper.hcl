server "couper" {
  files {
<<<<<<< HEAD
    document_root = env.DOC_DIR
  }
}

defaults {
  environment_variables = {
    DOC_DIR = "./"
=======
    document_root = "./"
>>>>>>> Changes by Marcels advice for simpler execution
  }
}
