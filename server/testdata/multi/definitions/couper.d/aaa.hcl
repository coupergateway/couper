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
    origin = "https://example.org"
  }

  basic_auth "BA" {
    user     = "User"
    password = "Pass"
  }
}
