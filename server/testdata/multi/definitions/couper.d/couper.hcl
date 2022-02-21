definitions {
  backend "Backend" {
    origin = "https://example.org"
  }

  backend "Added" {
    origin = "https://added.com"
  }

  basic_auth "BA" {
    user     = "USR"
    password = "PWD"
  }
}
