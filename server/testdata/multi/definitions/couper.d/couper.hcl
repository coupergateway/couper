definitions {
  backend "Backend" {
    origin = "https://example.org"
  }

  backend "Added" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/small"
  }

  basic_auth "BA" {
    user     = "USR"
    password = "PWD"
  }
}
