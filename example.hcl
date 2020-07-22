server "couperConnect" {
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
            access_control = ["AccessToken"]
            backend = "httpbin"
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
            path = "/anything/${to_upper(env.USER)}" #Optional and only if set, remove basePath+endpoint path
            description = "optional field"
            origin_address = "httpbin.org:443"
            origin_host = "httpbin.org"
            request {
                headers = {
                    X-Env-User = [env.USER]
                    X-Req-Header = [req.headers.X-Set-Me]
                    Authorization = ["Bearer ${req.cookies.AccessToken}"]
                    Cookie: []
                }
            }
        }
    }
}

definitions {
    jwt "AccessToken" {
    cookie = "AccessToken"

    // signature_algorithm = "RS256"
    key_file = "pubkey.pem"
    # x5c from auth0
    // key = "MIIC/zCCAeegAwIBAgIJKTPK12WWhWaJMA0GCSqGSIb3DQEBCwUAMB0xGzAZBgNVBAMTEndhb2lvLmV1LmF1dGgwLmNvbTAeFw0yMDAzMTMxNjA3MjFaFw0zMzExMjAxNjA3MjFaMB0xGzAZBgNVBAMTEndhb2lvLmV1LmF1dGgwLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALjDhC362yZR6MDBGuALhCZJqVML3dfvoCt5qhEe4cFOP1SfhUUnjEZpArJEMKncINyZmJSwQxPwevBS+UTiE+TjcinJZMeuALrI/87CZ2Fp0TkMkkyxv6X9e+VlgQQRE+7lbkMNm4wOLCHMXIdnWOm1zXCz962TYUplmJQwwijPtzBC0M0n+TMaDVbaCQLRD74uPzR2sJuB8h8ABCOYz2YnVu9aHkIe+7KYtPn1gsl6EjltJvzDac5dKxIa79VGojCf276EiNoS8Fej4VXtopLW3TUHVvwrR6MjaqGwzselC36rDUM4fULOcM9LCThCmB6VTTnBSGO524zOA7fieDMCAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQU5R7M7FMi1CwvN+xlw8rsoCEjnAkwDgYDVR0PAQH/BAQDAgKEMA0GCSqGSIb3DQEBCwUAA4IBAQA1+WH7yJVdnae6dW7t0IJ9b3hiy6GbJs3qF1hjIxPfynrMdPwQ9ong6UoISV8lwpK7rlVlgwF6peeYfbbYxl+4MnzTLECXjaYxdTsrzuB4AS4THeR4nU/1Mx0XQGt5xRij5/dtViVDBVGPQCieI/oU2UfOWnuFREiCMhGgPRNxT6P5lRUfa32rXiTiwRiSRDJA41xiWjdXxUY7lyNR/r+dRtpOufzFqtwHQ7KGMrzquygNvRcysJQyrxPNLmFXwTQB9NTcffSrmb0FIoe63pa958eoXmKSBgpT0DzfyFDZhKN6yZ27DV9ZMwBPgrwEzeqU5j4Epr8AnhnegEb3owi2"
    // key = "secret"
    
    signature_algorithm = "HS256"
    
    claims {
      iss = "TokenFactory"
      aud = "MyApp"
    }
  }
}
