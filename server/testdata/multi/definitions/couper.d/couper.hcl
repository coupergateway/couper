definitions {
  backend "Backend" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/small"
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
