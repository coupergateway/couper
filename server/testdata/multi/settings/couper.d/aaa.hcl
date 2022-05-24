defaults {
  environment_variables = {
    ZZZ = "/xyz"
  }
}

settings {
  health_path = "/def"

  request_id_client_header  = "Req-Id-Cl-Hdr"
  request_id_backend_header = "Req-Id-Be-Hdr"

  ca_file = "../../../integration/files/certificate.pem"
}
