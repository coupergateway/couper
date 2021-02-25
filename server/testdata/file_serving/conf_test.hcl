server "TestFileServing" {
  hosts = ["example.com"]

  base_path = "/apps/shiny-product"
  error_file = "./error.html"

  files {
    document_root = "./htdocs"
  }

  api {
    base_path = "/api"
    error_file = "./error.json"

    endpoint "/" {
      proxy {
        backend {
          origin = "{{.origin}}"
          hostname = "{{.hostname}}"
        }
      }
    }
  }

  spa {
    base_path = "/app"
    bootstrap_file = "./htdocs/spa.html"
    paths = ["/**"]
  }
}
