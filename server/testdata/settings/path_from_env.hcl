server {
  access_control = ["token"]
}

definitions {
  jwt "token" {
    key_file = env.KEY
    signature_algorithm = "HS256"
  }
}

defaults {
  environment_variables = {
    KEY = "public.pem"
  }
}
