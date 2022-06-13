server {
  access_control = ["token"]
}

definitions {
  jwt "token" {
    key_file = env.KEY_FILE
    signature_algorithm = "HS256"
  }
}

defaults {
  environment_variables = {
    KEY_FILE = "public.pem"
  }
}
