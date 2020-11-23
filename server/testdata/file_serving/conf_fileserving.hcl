server "fileserving-tests" {
    hosts = ["example.com"]

    error_file = "./error.html"

    files {
        document_root = "./htdocs"
    }

    spa {
        bootstrap_file = "./htdocs/spa.html"
        paths = [
            // files win
            "/dir/**",
            "/app/**",
            // api wins
            "/api/**",
            // spa wins
            "/"
        ]
    }

    api {
        base_path = "/api"
        error_file = "./error.json"
        endpoint "/foo/**" {
            backend {
                path = "/**"
                origin = "{{.origin}}"
                hostname = "{{.hostname}}"
            }
        }
    }
}
