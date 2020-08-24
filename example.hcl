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
        base_path = "/api/v1/${env.NOT_EXIST}"

        # reference backend definition
        backend = "my_proxy"

        endpoint "/timeout" {
            backend "my_proxy" {
                origin = "http://blackhole.webpagetest.org"
                # > timeout
                connect_timeout = "60s"
            }
        }

        endpoint "/connect-timeout" {
            backend "my_proxy" {
                origin = "http://1.2.3.4"
                connect_timeout = "2s"
            }
        }

        # pattern
        endpoint "/proxy/" {
        }

        endpoint "/filex/" {
            # inline backend definition
            backend {
                origin = "http://filex.github.io"
                hostname = "ferndrang.de"
                path = "/"
            }
        }

        endpoint "/httpbin/**" {
            backend "my_proxy" {
                path = "/**"

                request_headers = {
                    x-env-user = ["override-user"]
                    x-single-val = 12+14
                    user-agent = "moo"
                }

                response_headers = {
                    server = ["my-override"]
                    x-single-val = true
                }
            }
        }

        endpoint "/httpbin" {
            access_control = ["AccessToken"]
            backend = "httpbin"
        }
    }
}

definitions {
    backend "my_proxy" {
        origin = "https://couper.io:${442 + 1}"
        timeout = "20s"
        request_headers = {
            x-my-custom-ua = [req.headers.user-agent, false, to_upper("muh")]
            x-env-user = [env.USER]
        }
        
        response_headers = {
            Server = [to_lower("mySuperService")]
        }
    }

    backend "httpbin" {
        path = "/anything/" #Optional and only if set, remove basePath+endpoint path
        origin = "https://httpbin.org:443"
        request_headers = {
            X-Env-User = env.USER
            X-Req-Header = [req.headers.X-Set-Me]
            Authorization = ["Bearer ${req.cookies.AccessToken}"]
            Cookie: ""
        }
    }


    jwt "AccessToken" {
    header = "Authorization"

    // signature_algorithm = "RS256"
    key_file = "access_control/testdata/jwt/pubkey.pem"
    # x5c from auth0
    // key = "MIIC/zCCAeegAwIBAgIJKTPK12WWhWaJMA0GCSqGSIb3DQEBCwUAMB0xGzAZBgNVBAMTEndhb2lvLmV1LmF1dGgwLmNvbTAeFw0yMDAzMTMxNjA3MjFaFw0zMzExMjAxNjA3MjFaMB0xGzAZBgNVBAMTEndhb2lvLmV1LmF1dGgwLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALjDhC362yZR6MDBGuALhCZJqVML3dfvoCt5qhEe4cFOP1SfhUUnjEZpArJEMKncINyZmJSwQxPwevBS+UTiE+TjcinJZMeuALrI/87CZ2Fp0TkMkkyxv6X9e+VlgQQRE+7lbkMNm4wOLCHMXIdnWOm1zXCz962TYUplmJQwwijPtzBC0M0n+TMaDVbaCQLRD74uPzR2sJuB8h8ABCOYz2YnVu9aHkIe+7KYtPn1gsl6EjltJvzDac5dKxIa79VGojCf276EiNoS8Fej4VXtopLW3TUHVvwrR6MjaqGwzselC36rDUM4fULOcM9LCThCmB6VTTnBSGO524zOA7fieDMCAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQU5R7M7FMi1CwvN+xlw8rsoCEjnAkwDgYDVR0PAQH/BAQDAgKEMA0GCSqGSIb3DQEBCwUAA4IBAQA1+WH7yJVdnae6dW7t0IJ9b3hiy6GbJs3qF1hjIxPfynrMdPwQ9ong6UoISV8lwpK7rlVlgwF6peeYfbbYxl+4MnzTLECXjaYxdTsrzuB4AS4THeR4nU/1Mx0XQGt5xRij5/dtViVDBVGPQCieI/oU2UfOWnuFREiCMhGgPRNxT6P5lRUfa32rXiTiwRiSRDJA41xiWjdXxUY7lyNR/r+dRtpOufzFqtwHQ7KGMrzquygNvRcysJQyrxPNLmFXwTQB9NTcffSrmb0FIoe63pa958eoXmKSBgpT0DzfyFDZhKN6yZ27DV9ZMwBPgrwEzeqU5j4Epr8AnhnegEb3owi2"
    // key = "secret"
    
    signature_algorithm = "HS256"
    
    claims = {
        iss = "TokenFactory"
        aud = "MyApp"
        cpt = "hook"
    }
  }
}
