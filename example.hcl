server "couperConnect" {
    base_path = "/api/v1/"

    files {
        document_root = "./public"
    }

    # pattern
    path "/proxy/" {
        # reference backend definition
        backend = "my_proxy"
    }

    path "/filex/" {
        # inline backend definition
        backend "proxy" "upstream" { #kind with reserved keyword 'proxy'
            origin_address = "filex.github.io:80"
            origin_host = "ferndrang.de"
            path = "/"
        }
    }

    path "/httpbin/" {
        backend = "httpbin"
    }

    backend "proxy" "my_proxy" {
        description = "you could reference me with path blocks"
        origin_address = "couper.io:${442 + 1}"
        origin_host = "couper.io"
        request {
            headers = {
                X-My-Custom-Foo-UA = [req.headers.User-Agent, to_upper("muh")]
                X-Env-User = [env.USER]
            }
        }

        response {
            headers = {
                Server = [to_lower("mySuperService")]
            }
        }
    }

    backend "proxy" "httpbin" {
        path = "/headers" #Optional and only if set, remove basePath+endpoint path
        description = "optional field"
        origin_address = "httpbin.org:443"
        origin_host = "httpbin.org"
        request {
            headers = {
                X-Env-User = [env.USER]
                X-Req-Header = [req.headers.X-Set-Me]
            }
        }
    }
}
