application "couperConnect" {
    base_path = "/api/v1/"

    files {
        document_root = "./public"
    }

    # pattern, kind or #ref
    path "/proxy/" "my_proxy" {} #ref

    path "/filex/" "proxy" { #kind with reserved keyword 'proxy'
        origin_address = "filex.github.io:80"
        origin_host = "ferndrang.de"
    }

    path "/httpbin/" "httpbin" {} #original 'httpbin' settings

    path "/httpbin2/" "httpbin" { #override 'httpbin' settings <-- not working
        request {
            headers = {
                X-Env-User = ["couper"]
            }
        }
    }

    backend "proxy" "my_proxy" {
        description = "you could reference me with path blocks"
        origin_address = "couper.io:${442 + 1}"
        origin_host = "couper.io"
        request {
            headers = {
                X-My-Custom-Foo-UA = ["ua:${req.headers.User-Agent}", "muh"]
                X-Env-User = ["${env.USER}"]
            }
        }

        response {
            headers = {
                Server = ["mySuperService"]
            }
        }
    }

    backend "proxy" "httpbin" {
        path = "/headers/" #Optional and only if set, remove basePath+endpoint path
        description = "optional field"
        origin_address = "httpbin.org:443"
        origin_host = "httpbin.org"
        request {
            headers = {
                X-Env-User = ["${env.USER}"]
                X-Req-Header = ["${req.headers.X-Set-Me}"]
            }
        }
    }
}
