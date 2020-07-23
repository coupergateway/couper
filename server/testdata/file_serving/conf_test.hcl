server "TestFileServing" {
  domains = ["example.com"]

  files {
    document_root = "testdata/file_serving/htdocs"
    error_file = "testdata/file_serving/error.html"
  }

  base_path = "/apps/shiny-product"

  api {
    base_path = "/api"

    endpoint "/" {
      backend {
        origin = "{{.origin}}"
        hostname = "{{.hostname}}"
      }
    }
  }

  spa {
    base_path = "/app"
    bootstrap_file = "testdata/file_serving/htdocs/spa.html"
    paths = ["/**"]
  }
}