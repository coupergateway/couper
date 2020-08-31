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
                    x-uuid = req.id
                }
            }
        }

        endpoint "/httpbin" {
            access_control = ["jwtio"]
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
        path = "/anything" #Optional and only if set, remove basePath+endpoint path
        origin = "https://httpbin.org:443"
        request_headers = {
            X-Env-User = env.USER
            X-Claims = [
                req.ctx.jwtio.admin,
                req.ctx.jwtio.aud,
                req.ctx.jwtio.iat,
                req.ctx.jwtio.name,
                req.ctx.jwtio.sub,
            ]
            x-vars = [req.id, req.method, req.endpoint, req.path, req.url]
        }

        response_headers = {
            uuid = req.id
            bereq-path = bereq.path
            status = beresp.status
        }
    }

    jwt "jwtio" {
        header = "Authorization"
        signature_algorithm = "RS512"
        key = "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUlJQklqQU5CZ2txaGtpRzl3MEJBUUVGQUFPQ0FROEFNSUlCQ2dLQ0FRRUFuenlpczFaamZOQjBiQmdLRk1Tdgp2a1R0d2x2QnNhSnE3UzV3QStremVWT1ZwVld3a1dkVmhhNHMzOFhNL3BhL3lyNDdhdjcrejNWVG12RFJ5QUhjCmFUOTJ3aFJFRnBMdjljajVsVGVKU2lieXIvTXJtL1l0akNaVldnYU9ZSWh3clh3S0xxUHIvMTFpbldzQWtmSXkKdHZIV1R4WllFY1hMZ0FYRnVVdWFTM3VGOWdFaU5Rd3pHVFUxdjBGcWtxVEJyNEI4blczSENONDdYVXUwdDhZMAplK2xmNHM0T3hRYXdXRDc5SjkvNWQzUnkwdmJWM0FtMUZ0R0ppSnZPd1JzSWZWQ2hEcFlTdFRjSFRDTXF0dldiClY2TDExQldrcHpHWFNXNEh2NDNxYStHU1lPRDJRVTY4TWI1OW9TazJPQitCdE9McEpvZm1iR0VHZ3Ztd3lDSTkKTXdJREFRQUIKLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg=="

        claims = {
            aud     = "one"
            admin   = true
            iat     = 1516239022
            name    = base64_decode("Sm9obiBEb2U=")
            sub     = "1234567890"
        }

        required_claims = ["name"]
    }
}
