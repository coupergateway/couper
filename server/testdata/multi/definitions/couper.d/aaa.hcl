server {
  endpoint "/added" {
    access_control = ["BA"]

    proxy {
      backend = "Added"
    }
  }
}

definitions {
  backend "Backend" {
    origin = "${env.COUPER_TEST_BACKEND_ADDR}"
    path = "/small"
  }

  basic_auth "BA" {
    user     = "User"
    password = "Pass"
  }
}
