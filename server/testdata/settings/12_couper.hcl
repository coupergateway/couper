server {
  files {
    document_root = "./"

    set_response_headers = {
      test-key = "value"
      Test-Key = "value"
    }
  }
}
