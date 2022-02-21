definitions {
  backend "Backend" {
    origin = "https://httpbin.org"
  }

  basic_auth "BA" {
    user     = "User"
    password = "Pass"
  }
}
