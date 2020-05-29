frontend "couperConnect" {
    base_path = "/api/v1/"
    endpoint "/proxy/" {
        backend "Proxy" {
            description = "optional field"
            origin_address = "couper.io:${442 + 1}"
            origin_host = "couper.io"
            request_headers = {
                X-My-Custom-Foo = ["ua:$//{req.http.user-agent}", "muh"]
                X-Env-User = ["${env.USER}"]
            }
        }
    }   
}

