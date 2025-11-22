server "acs" {
  access_control = ["ba1"]
  error_file = "../api_error.json"
  api {
    base_path = "/v1"
    disable_access_control = ["ba1"]
    endpoint "/**" {
      # access_control = ["ba1"] # not possible atm TODO: spec
      proxy {
        backend "test" {
          set_request_headers = {
            auth = ["ba1"]
          }
        }
      }
    }
  }

  api {
    base_path = "/v2"
    access_control = ["ba2"]
    endpoint "/**" {
      proxy {
        backend "test" {
          set_request_headers = {
            auth = ["ba1", "ba2"]
            Authorization = request.headers.authorization # proxy blacklist
          }
        }
      }
    }
  }

  api {
    base_path = "/v3"
    access_control = ["ba2"]
    endpoint "/**" {
      access_control = ["ba3"]
      disable_access_control = ["ba1", "ba2", "ba3"]
      proxy {
        backend = "test"
      }
    }
  }

  api {
    base_path = "/v4"
    access_control = ["ba2"]
    endpoint "/**" {
      error_file = "../server_error.html" # error_file in endpoint
      proxy {
        backend = "test"
      }
    }
  }

  api {
    base_path = "/v5"
    access_control = ["ba2"]
    cors {
      allowed_origins = ["*"]
    }
    endpoint "/exists" {
      response {
        body = "exists"
      }
    }
  }

  endpoint "/status" {
    disable_access_control = ["ba1"]
    proxy {
      backend = "test"
    }
  }

  endpoint "/superadmin" {
    access_control = ["ba4"]
    proxy {
      backend "test" {
        set_request_headers = {
          auth = ["ba1", "ba4"]
          Authorization = request.headers.authorization # proxy blacklist
        }
      }
    }
  }

  endpoint "/ba5" {
    access_control = ["ba5"]
    disable_access_control = ["ba1"]
    proxy {
      backend "test" {
        set_request_headers = {
          X-Ba-User = request.context.ba5.user
          Authorization = request.headers.authorization # proxy blacklist
        }
        set_response_headers = {
          X-BA-User = request.context.ba5.user
        }
      }
    }
  }

  endpoint "/jwt" {
    disable_access_control = ["ba1"]
    access_control = ["JWTToken"]
    response {
      headers = {
        x-jwt-sub = request.context.JWTToken.sub
        x-granted-permissions = json_encode(request.context.granted_permissions)
      }
    }
  }

  endpoint "/jwt/dpop" {
    disable_access_control = ["ba1"]
    access_control = ["JWTTokenDPoP"]
    response {}
  }

  endpoint "/jwt/cookie" {
    disable_access_control = ["ba1"]
    access_control = ["JWTTokenCookie"]
    response {}
  }

  endpoint "/jwt/header" {
    disable_access_control = ["ba1"]
    access_control = ["JWTTokenHeader"]
    response {}
  }

  endpoint "/jwt/header/auth" {
    disable_access_control = ["ba1"]
    access_control = ["JWTTokenHeaderAuth"]
    response {}
  }

  endpoint "/jwt/tokenValue" {
    disable_access_control = ["ba1"]
    access_control = ["JWTTokenTokenValue"]
    response {}
  }

  endpoint "/jwt/token_value_query" {
    disable_access_control = ["ba1"]
    access_control = ["JWT_token_value_query"]
    response {
      headers = {
        x-jwt-sub = request.context.JWT_token_value_query.sub
        x-granted-permissions = json_encode(request.context.granted_permissions)
      }
    }
  }

  endpoint "/jwt/token_value_body" {
    disable_access_control = ["ba1"]
    access_control = ["JWT_token_value_body"]
    response {
      headers = {
        x-jwt-sub = request.context.JWT_token_value_body.sub
        x-granted-permissions = json_encode(request.context.granted_permissions)
      }
    }
  }

  endpoint "/jwt/rsa" {
    disable_access_control = ["ba1"]
    access_control = ["RSAToken"]
    response {
      headers = {
        x-jwt-sub = request.context.RSAToken.sub
      }
    }
  }

  endpoint "/jwt/rsa/pkcs1" {
    disable_access_control = ["ba1"]
    access_control = ["RSAToken1"]
    response {
      headers = {
        x-jwt-sub = request.context.RSAToken1.sub
      }
    }
  }

  endpoint "/jwt/rsa/pkcs8" {
    disable_access_control = ["ba1"]
    access_control = ["RSAToken8"]
    response {
      headers = {
        x-jwt-sub = request.context.RSAToken8.sub
      }
    }
  }

  endpoint "/jwt/rsa/bad" {
    disable_access_control = ["ba1"]
    access_control = ["RSATokenWrongAlgorithm"]
    response {
      headers = {
        x-jwt-sub = request.context.RSATokenWrongAlgorithm.sub
      }
    }
  }

  endpoint "/jwt/ecdsa" {
    disable_access_control = ["ba1"]
    access_control = ["ECDSAToken"]
    response {
      headers = {
        x-jwt-sub = request.context.ECDSAToken.sub
      }
    }
  }

  endpoint "/jwt/ecdsa8" {
    disable_access_control = ["ba1"]
    access_control = ["ECDSAToken8"]
    response {
      headers = {
        x-jwt-sub = request.context.ECDSAToken8.sub
      }
    }
  }

  endpoint "/jwt/ecdsa/bad" {
    disable_access_control = ["ba1"]
    access_control = ["ECDSATokenWrongAlgorithm"]
    response {
      headers = {
        x-jwt-sub = request.context.ECDSATokenWrongAlgorithm.sub
      }
    }
  }

  endpoint "/jwks/rsa" {
    disable_access_control = ["ba1"]
    access_control = ["JWKS"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKS.sub
      }
    }
  }

  endpoint "/jwks/ecdsa" {
    disable_access_control = ["ba1"]
    access_control = ["JWKS"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKS.sub
      }
    }
  }

  endpoint "/jwks/rsa/scope" {
    disable_access_control = ["ba1"]
    access_control = ["JWKS_scope"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKS_scope.sub
        x-granted-permissions = json_encode(request.context.granted_permissions)
      }
    }
  }

  endpoint "/jwks/rsa/not_found" {
    disable_access_control = ["ba1"]
    access_control = ["JWKS_not_found"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKS_not_found.sub
      }
    }
  }

  endpoint "/jwks/rsa/remote" {
    disable_access_control = ["ba1"]
    access_control = ["JWKSRemote"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKSRemote.sub
      }
    }
  }
  endpoint "/jwks/rsa/backend" {
    disable_access_control = ["ba1"]
    access_control = ["JWKSBackend"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKSBackend.sub
      }
    }
  }
  endpoint "/jwks/rsa/backendref" {
    disable_access_control = ["ba1"]
    access_control = ["JWKSBackendRef"]
    response {
      headers = {
        x-jwt-sub = request.context.JWKSBackendRef.sub
      }
    }
  }
  endpoint "/jwt/create" {
    disable_access_control = ["ba1"]
    response {
      body = jwt_sign(request.query.type[0], {"sub":1234567890})
    }
  }

  endpoint "/rate1/**" {
    disable_access_control = ["ba1"]
    access_control = ["jwt_rate", "rate"]
    response {
      json_body = {
        ok = true
      }
    }
  }
  endpoint "/rate2/**" {
    disable_access_control = ["ba1"]
    access_control = ["jwt_rate", "rate"]
    response {
      json_body = {
        ok = true
      }
    }
  }
  endpoint "/rate3/**" {
    disable_access_control = ["ba1"]
    access_control = ["rate_eh"]
    response {
      json_body = {
        ok = true
      }
    }
  }
}

definitions {
  basic_auth "ba1" {
    password = "asdf"
  }
  basic_auth "ba2" {
    password = "asdf"
  }
  basic_auth "ba3" {
    password = "asdf"
  }
  basic_auth "ba4" {
    password = "asdf"
  }
  basic_auth "ba5" {
    user     = "USR"
    password = "PWD"
  }
  jwt "JWTToken" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    permissions_claim = "scope"
  }
  jwt "JWTTokenDPoP" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    beta_dpop = true
  }
  jwt "JWTTokenCookie" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    cookie = "tok"
  }
  jwt "JWTTokenHeader" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    header = "x-token"
  }
  jwt "JWTTokenHeaderAuth" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    header = "aUtHoRiZaTiOn"
  }
  jwt "JWTTokenTokenValue" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    token_value = request.query.tok[0]
  }
  jwt "RSAToken" {
    signature_algorithm = "RS256"
    key_file = "../files/certificate.pem"
  }
  jwt "RSAToken1" {
    signature_algorithm = "RS256"
    bearer = true # default
    key =<<-EOF
        -----BEGIN RSA PUBLIC KEY-----
        MIIBCgKCAQEAxOubq8QN8gBVEwINCfVNvmZAhO+ZLeKZapT38OyZkqm+8BUs98cB
        FmzUCiuN2cFrjuhoRAXj2YV/3lu0Sy/G3knLFbGSfuJ+oZuwYNDA3lasGJNZonRE
        sAUJde1hI0uJbceJzcJDifUx2zGR5eCRQKlxxiV/irEy+wZ+/fN9xrue18BykLz6
        HQBXu4mhc17q9qAZtx3hLBRxQwkZGbxumgYGdPXuh2YV82adw18wiZIXgVOvawgX
        QvlVDnjSaLqE3RE/bkVmWkE4TRQuFYhqoEFV50RBILEWlwUHqNggL9zUw2/RdW1u
        TyQJtEMRiz6WgiWaq0l9SkmlrSFA2SDA5wIDAQAB
        -----END RSA PUBLIC KEY-----
    EOF
  }
  jwt "RSAToken8" {
    header = "Authorization" # keep for now
    signature_algorithm = "RS256"
    key =<<-EOF
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxOubq8QN8gBVEwINCfVN
        vmZAhO+ZLeKZapT38OyZkqm+8BUs98cBFmzUCiuN2cFrjuhoRAXj2YV/3lu0Sy/G
        3knLFbGSfuJ+oZuwYNDA3lasGJNZonREsAUJde1hI0uJbceJzcJDifUx2zGR5eCR
        QKlxxiV/irEy+wZ+/fN9xrue18BykLz6HQBXu4mhc17q9qAZtx3hLBRxQwkZGbxu
        mgYGdPXuh2YV82adw18wiZIXgVOvawgXQvlVDnjSaLqE3RE/bkVmWkE4TRQuFYhq
        oEFV50RBILEWlwUHqNggL9zUw2/RdW1uTyQJtEMRiz6WgiWaq0l9SkmlrSFA2SDA
        5wIDAQAB
        -----END PUBLIC KEY-----
    EOF
  }
  jwt "RSATokenWrongAlgorithm" {
    signature_algorithm = "RS384"
    key_file = "../files/certificate.pem"
  }
  jwt "ECDSAToken" {
    signature_algorithm = "ES256"
    key_file = "../files/certificate-ecdsa.pem"
    signing_ttl = "10s"
    signing_key_file = "../files/ecdsa.key"
  }
  jwt "ECDSAToken8" {
    signature_algorithm = "ES256"
    key =<<-EOF
        -----BEGIN PUBLIC KEY-----
        MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEgPxsi3Y2J1FWrjXjacAWmbB+GIuz
        KPLrW5KikaxLtwuoDE61oaWMM4H99mGPN7k4Bmamle8ne9Pr7rQhXuk8Iw==
        -----END PUBLIC KEY-----
    EOF
  }

  jwt "ECDSATokenWrongAlgorithm" {
    signature_algorithm = "ES384"
    key_file = "../files/certificate-ecdsa.pem"
  }

  jwt "JWKS" {
    jwks_url = "file:../files/jwks.json"
  }

  jwt "JWKS_scope" {
    jwks_url = "file:../files/jwks.json"
    permissions_claim = "scope"
  }

  jwt "JWKSRemote" {
    jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
  }

  jwt "JWKS_not_found" {
    jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/not.found"
  }

  jwt "JWKSBackend" {
    jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
    backend {
      origin = env.COUPER_TEST_BACKEND_ADDR
    }
  }

  jwt "JWKSBackendRef" {
    jwks_url = "${env.COUPER_TEST_BACKEND_ADDR}/jwks.json"
    backend = "jwks"
  }

  jwt "JWT_token_value_query" {
    token_value = request.query.token[0]
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    permissions_claim = "scope"
  }

  jwt "JWT_token_value_body" {
    token_value = request.json_body.token
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
    permissions_claim = "scope"
  }

  jwt "jwt_rate" {
    signature_algorithm = "HS256"
    key = "y0urS3cretT08eU5edF0rC0uPerInThe3xamp1e"
  }
  beta_rate_limiter "rate" {
    period = "100s"
    per_period = 1
    # period_window = "fixed"
    key = request.context.jwt_rate.sub
  }
  beta_rate_limiter "rate_eh" {
    period = "1s"
    per_period = 1
    key = "asdf"
    error_handler "beta_rate_limiter" {
      response {
        status = 418
      }
    }
  }

  backend "jwks" {
    origin = env.COUPER_TEST_BACKEND_ADDR
  }

  backend "test" {
    origin = env.COUPER_TEST_BACKEND_ADDR
    path = "/anything"
    set_request_headers = {
      Authorization = request.headers.authorization
    }
  }
}

settings {
  no_proxy_from_env = true
}
