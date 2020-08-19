server "fileserving-tests" {
    //fixme: make optional
    listen = ["example.com:443"]
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
            "/api/**"
        ]
    }

    api {
        base_path = "/api"
        endpoint "/foo/**" {
            backend {
                path = "/**"
                origin = "{{.origin}}"
                hostname = "{{.hostname}}"
            }
        }
    }
}
