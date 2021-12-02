server {
  files {
    document_root = "./"

    set_response_headers = {
      test-key = "value"
      test-key = "value"
    }
  }
}
