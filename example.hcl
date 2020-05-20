frontend "couperConnect" {
    base_path = "/api/v1/"
    endpoint "/proxy/" {
        backend "Proxy" {
            origin_address = "couper.io:443"
            origin_host = "couper.io"
            // Headers = {
            //     "X-Myproxy-Header" = "${req.x-request-id}"
            // }
        }
        
    }
}
