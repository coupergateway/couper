server "fileserving-tests" {
    hosts = ["example.com:0"]

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
            "/",
        ]
    }

    spa "myapp" {
        bootstrap_file = "./htdocs/my_app/spa.html"
        paths = [
            "/my_app",
            "/my_app/**"
        ]
        bootstrap_data = {
            framework: "react.js"
        }
    }

    api {
        base_path = "/api"
        error_file = "./error.json"
        endpoint "/foo/**" {
            proxy {
                backend {
                    path = "/**"
                    origin = "{{.origin}}"
                    hostname = "test.couper.io"
                }
            }
        }
    }
}
