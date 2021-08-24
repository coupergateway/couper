server "couper" {
<<<<<<< HEAD
<<<<<<< HEAD
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
=======
  # --> type: couper_access
  endpoint "/" {
    proxy {
//      url = "https://httpbin.org/anything" # --> backend
      backend { # --> type: couper_backend
        origin = "https://httpbin.org"
        path = "/anything"
      }
    }
>>>>>>> Implemented changes for uniform Log-Format fields
=======
  files {
    document_root = "/htdocs"
>>>>>>> Adjusted changelog, reset changes in couper.hcl
  }
}
