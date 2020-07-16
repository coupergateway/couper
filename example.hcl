server "couperConnect" {
    domains = ["127.0.0.1"]

    files {
        document_root = "./public"
    }

    spa {
        bootstrap_file = "./public/bs.html"
        paths = [
            "/app/foo",
            "/app/bar",
        ]
    }

    api {
        base_path = "/api/v1/"

        # pattern
        endpoint "/proxy/" {
            # reference backend definition
            backend = "my_proxy"
        }

        endpoint "/filex/" {
            # inline backend definition
            backend "proxy" { #kind with reserved keyword 'proxy'
                origin_address = "filex.github.io:80"
                origin_host = "ferndrang.de"
                path = "/"
            }
        }

        endpoint "/httpbin/**" {
            backend "proxy" {
                origin_address = "httpbin.org:443"
                origin_host = "httpbin.org"
                path = "/**"
            }
        }

        endpoint "/httpbin" {
            backend = "httpbin"
        }

        endpoint "/status/{status:[0-9]{3}}" {
            backend "proxy" {
                origin_address = "httpbin.org:443"
                origin_host = "httpbin.org"
                path = "/status/${req.params.status}"
                request {
                    headers = {
                        X-Status = [req.params.status]
                    }
                }
            }
        }

        backend "proxy" "my_proxy" {
            description = "you could reference me with endpoint blocks"
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

}
